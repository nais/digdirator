package common

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"strings"
	"sync"
	"time"

	"github.com/google/uuid"
	naisiov1 "github.com/nais/liberator/pkg/apis/nais.io/v1"
	libfinalizer "github.com/nais/liberator/pkg/finalizer"
	"github.com/nais/liberator/pkg/kubernetes"
	log "github.com/sirupsen/logrus"
	"gopkg.in/square/go-jose.v2"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/tools/record"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"

	"github.com/nais/digdirator/pkg/clients"
	"github.com/nais/digdirator/pkg/config"
	"github.com/nais/digdirator/pkg/crypto"
	"github.com/nais/digdirator/pkg/digdir"
	"github.com/nais/digdirator/pkg/digdir/types"
	"github.com/nais/digdirator/pkg/metrics"
)

const RequeueInterval = 10 * time.Second

type Reconciler struct {
	Client     client.Client
	Reader     client.Reader
	Scheme     *runtime.Scheme
	Recorder   record.EventRecorder
	Config     *config.Config
	Signer     jose.Signer
	HttpClient *http.Client
	ClientID   []byte
}

func NewReconciler(
	client client.Client,
	reader client.Reader,
	scheme *runtime.Scheme,
	recorder record.EventRecorder,
	config *config.Config,
	signer jose.Signer,
	httpClient *http.Client,
	clientID []byte,
) Reconciler {
	return Reconciler{
		Client:     client,
		Reader:     reader,
		Scheme:     scheme,
		Recorder:   recorder,
		Config:     config,
		Signer:     signer,
		HttpClient: httpClient,
		ClientID:   clientID,
	}
}

func (r *Reconciler) Reconcile(ctx context.Context, req ctrl.Request, instance clients.Instance) (ctrl.Result, error) {
	tx, err := r.prepare(ctx, req, instance)
	if err != nil {
		return ctrl.Result{}, client.IgnoreNotFound(err)
	}

	defer func() {
		tx.Logger.Infof("finished processing request")
	}()

	finalizer := r.finalizer(tx)

	if libfinalizer.IsBeingDeleted(tx.Instance) {
		return finalizer.Process()
	}

	if !libfinalizer.HasFinalizer(tx.Instance, FinalizerName) {
		return finalizer.Register()
	}

	if isUpToDate, err := clients.IsUpToDate(tx.Instance); isUpToDate {
		if err != nil {
			return ctrl.Result{}, err
		}
		tx.Logger.Info("object state already reconciled, nothing to do")
		return ctrl.Result{}, nil
	}

	err = r.process(tx)
	if err != nil {
		return r.handleError(tx, err)
	}

	return r.complete(tx)
}

func (r *Reconciler) prepare(ctx context.Context, req ctrl.Request, instance clients.Instance) (*Transaction, error) {
	instanceType := clients.GetInstanceType(instance)
	correlationID := uuid.New().String()

	logger := *log.WithFields(log.Fields{
		"instance_type":      instanceType,
		"instance_name":      req.Name,
		"instance_namespace": req.Namespace,
		"correlationID":      correlationID,
	})

	if err := r.Reader.Get(ctx, req.NamespacedName, instance); err != nil {
		return nil, err
	}
	instance.GetStatus().SetCorrelationID(correlationID)

	digdirClient := digdir.NewClient(r.HttpClient, r.Signer, r.Config, instance, r.ClientID)

	logger.Infof("processing %s...", instanceType)

	transaction := NewTransaction(
		ctx,
		instance,
		&logger,
		&digdirClient,
		r.Config,
	)
	return &transaction, nil
}

func (r *Reconciler) process(tx *Transaction) error {
	instanceClient, err := r.createOrUpdateClient(tx)
	if err != nil {
		return err
	}

	if len(tx.Instance.GetStatus().GetClientID()) == 0 && instanceClient != nil {
		tx.Instance.GetStatus().SetClientID(instanceClient.ClientID)
	}

	secretsClient := r.secrets(tx)
	managedSecrets, err := secretsClient.GetManaged()
	if err != nil {
		return fmt.Errorf("getting managed secrets: %w", err)
	}

	var jwk *jose.JSONWebKey

	if clients.NeedsSecretRotation(tx.Instance) {
		jwk, err = crypto.GenerateJwk()
		if err != nil {
			return fmt.Errorf("generating new JWK for client: %w", err)
		}

		if err := r.registerJwk(tx, *jwk, *managedSecrets, instanceClient.ClientID); err != nil {
			return err
		}

		r.reportEvent(tx, corev1.EventTypeNormal, EventRotatedInDigDir, "Client credentials is rotated")
		metrics.IncClientsRotated(tx.Instance)
	} else {
		jwk, err = crypto.GetPreviousJwkFromSecret(managedSecrets, clients.GetSecretJwkKey(tx.Instance))
		if err != nil {
			return err
		}
	}

	if err := secretsClient.CreateOrUpdate(*jwk); err != nil {
		return fmt.Errorf("creating or updating secret: %w", err)
	}

	if err := secretsClient.DeleteUnused(managedSecrets.Unused); err != nil {
		return err
	}

	return nil
}

func (r *Reconciler) createOrUpdateClient(tx *Transaction) (*types.ClientRegistration, error) {
	registration, err := tx.DigdirClient.GetRegistration(tx.Instance, tx.Ctx, tx.Config.ClusterName)
	if err != nil {
		return nil, fmt.Errorf("checking if client exists: %w", err)
	}

	registrationPayload := clients.ToClientRegistration(tx.Instance, tx.Config)

	switch instance := tx.Instance.(type) {
	case *naisiov1.MaskinportenClient:
		exposedScopes := instance.GetExposedScopes()
		scopes := r.scopes(tx)

		err := scopes.Process(exposedScopes)
		if err != nil {
			return nil, fmt.Errorf("processing scopes: %w", err)
		}

		filteredPayload, err := r.filterValidScopes(tx, registrationPayload)
		if err != nil {
			return nil, err
		}

		registrationPayload = *filteredPayload
	}

	if registration != nil {
		_, err = r.updateClient(tx, registrationPayload, registration.ClientID)
		if err != nil {
			return nil, fmt.Errorf("updating client: %w", err)
		}

		r.reportEvent(tx, corev1.EventTypeNormal, EventUpdatedInDigDir, "Client is updated")
		metrics.IncClientsUpdated(tx.Instance)
	} else {
		registration, err = r.createClient(tx, registrationPayload)
		if err != nil {
			return nil, fmt.Errorf("creating client: %w", err)
		}

		r.reportEvent(tx, corev1.EventTypeNormal, EventCreatedInDigDir, "Client is registered")
		metrics.IncClientsCreated(tx.Instance)
	}

	return registration, nil
}

func (r *Reconciler) createClient(tx *Transaction, payload types.ClientRegistration) (*types.ClientRegistration, error) {
	tx.Logger.Debug("client does not exist in Digdir, registering...")

	registrationResponse, err := tx.DigdirClient.Register(tx.Ctx, payload)
	if err != nil {
		return nil, fmt.Errorf("registering client to Digdir: %w", err)
	}

	tx.Logger = tx.Logger.WithField("ClientID", registrationResponse.ClientID)
	tx.Logger.Info("client registered")
	return registrationResponse, nil
}

func (r *Reconciler) updateClient(tx *Transaction, payload types.ClientRegistration, clientID string) (*types.ClientRegistration, error) {
	tx.Logger = tx.Logger.WithField("ClientID", clientID)
	tx.Logger.Debug("client already exists in Digdir, updating...")

	registrationResponse, err := tx.DigdirClient.Update(tx.Ctx, payload, clientID)
	if err != nil {
		return nil, fmt.Errorf("updating client at Digdir: %w", err)
	}

	tx.Logger.WithField("ClientID", registrationResponse.ClientID).Info("client updated")
	return registrationResponse, err
}

func (r *Reconciler) filterValidScopes(tx *Transaction, registration types.ClientRegistration) (*types.ClientRegistration, error) {
	var desiredScopes []naisiov1.ConsumedScope

	switch v := tx.Instance.(type) {
	case *naisiov1.IDPortenClient:
		return &registration, nil
	case *naisiov1.MaskinportenClient:
		desiredScopes = v.Spec.Scopes.ConsumedScopes
	}

	accessibleScopes, err := tx.DigdirClient.GetAccessibleScopes(tx.Ctx)
	if err != nil {
		return nil, err
	}

	if len(desiredScopes) == 0 {
		desiredScopes = []naisiov1.ConsumedScope{{Name: r.Config.DigDir.Maskinporten.Default.ClientScope}}
	}

	filteredScopes := clients.FilterScopes(desiredScopes, accessibleScopes)
	registration.Scopes = filteredScopes.Valid

	if len(filteredScopes.Invalid) > 0 {
		for _, scope := range filteredScopes.Invalid {
			msg := fmt.Sprintf("ERROR: Precondition failed: This organization number has not been granted access to the scope '%s'.", scope)
			tx.Logger.Error(msg)
			r.reportEvent(tx, corev1.EventTypeWarning, EventSkipped, msg)
		}
	}

	return &registration, nil
}

func (r *Reconciler) registerJwk(tx *Transaction, jwk jose.JSONWebKey, managedSecrets kubernetes.SecretLists, clientID string) error {
	jwks, err := crypto.MergeJwks(jwk, managedSecrets.Used, clients.GetSecretJwkKey(tx.Instance))
	if err != nil {
		return fmt.Errorf("merging new JWK with JWKs in use: %w", err)
	}

	tx.Logger.Debug("generated new JWKS for client, registering...")

	jwksResponse, err := tx.DigdirClient.RegisterKeys(tx.Ctx, clientID, jwks)

	if err != nil {
		return fmt.Errorf("registering JWKS for client at Digdir: %w", err)
	}

	tx.Instance.GetStatus().SetKeyIDs(crypto.KeyIDsFromJwks(&jwksResponse.JSONWebKeySet))
	tx.Logger = tx.Logger.WithField("KeyIDs", strings.Join(tx.Instance.GetStatus().GetKeyIDs(), ", "))
	tx.Logger.Info("new JWKS for client registered")

	return nil
}

func (r *Reconciler) handleError(tx *Transaction, err error) (ctrl.Result, error) {
	tx.Logger.Error(fmt.Errorf("processing client: %w", err))
	r.reportEvent(tx, corev1.EventTypeWarning, EventFailedSynchronization, "Failed to synchronize client")

	var digdirErr *digdir.Error
	requeue := true

	if errors.As(err, &digdirErr) {
		if errors.Is(err, digdir.ServerError) {
			r.reportEvent(tx, corev1.EventTypeNormal, EventRetrying, digdirErr.Message)
		} else if errors.Is(err, digdir.ClientError) {
			r.reportEvent(tx, corev1.EventTypeWarning, EventSkipped, digdirErr.Message)
			requeue = false
		}
	}

	if requeue {
		metrics.IncClientsFailedProcessing(tx.Instance)
		tx.Logger.Infof("requeuing failed reconciliation after %s", RequeueInterval)
		return ctrl.Result{RequeueAfter: RequeueInterval}, nil
	}

	return ctrl.Result{}, nil
}

func (r *Reconciler) complete(tx *Transaction) (ctrl.Result, error) {
	tx.Logger.Debugf("updating status for %s", clients.GetInstanceType(tx.Instance))

	hash, err := tx.Instance.Hash()
	if err != nil {
		return ctrl.Result{}, err
	}
	tx.Instance.GetStatus().SetHash(hash)

	tx.Instance.GetStatus().SetStateSynchronized()
	tx.Instance.GetStatus().SetSynchronizationSecretName(clients.GetSecretName(tx.Instance))

	if err := r.updateInstance(tx.Ctx, tx.Instance, func(existing clients.Instance) error {
		existing.SetStatus(*tx.Instance.GetStatus())
		return r.Client.Status().Update(tx.Ctx, existing)
	}); err != nil {
		r.reportEvent(tx, corev1.EventTypeWarning, EventFailedStatusUpdate, "Failed to update status")
		return ctrl.Result{}, fmt.Errorf("updating status subresource: %w", err)
	}

	tx.Logger.Info("status subresource successfully updated")

	r.reportEvent(tx, corev1.EventTypeNormal, EventSynchronized, "Client is up-to-date")
	tx.Logger.Info("successfully reconciled")
	metrics.IncClientsProcessed(tx.Instance)

	return ctrl.Result{}, nil
}

func (r *Reconciler) reportEvent(tx *Transaction, eventType, event, message string) {
	tx.Instance.GetStatus().SetSynchronizationState(event)
	r.Recorder.Event(tx.Instance, eventType, event, message)
}

var instancesync sync.Mutex

func (r *Reconciler) updateInstance(ctx context.Context, instance clients.Instance, updateFunc func(existing clients.Instance) error) error {
	instancesync.Lock()
	defer instancesync.Unlock()

	existing := func(instance clients.Instance) clients.Instance {
		switch instance.(type) {
		case *naisiov1.IDPortenClient:
			return &naisiov1.IDPortenClient{}
		case *naisiov1.MaskinportenClient:
			return &naisiov1.MaskinportenClient{}
		}
		return nil
	}(instance)

	err := r.Reader.Get(ctx, client.ObjectKey{Namespace: instance.GetNamespace(), Name: instance.GetName()}, existing)
	if err != nil {
		return fmt.Errorf("get newest version of %s: %s", clients.GetInstanceType(instance), err)
	}

	return updateFunc(existing)
}
