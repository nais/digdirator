package common

import (
	"context"
	"fmt"
	"github.com/google/uuid"
	v1 "github.com/nais/digdirator/api/v1"
	"github.com/nais/digdirator/pkg/config"
	"github.com/nais/digdirator/pkg/crypto"
	"github.com/nais/digdirator/pkg/digdir"
	"github.com/nais/digdirator/pkg/digdir/types"
	"github.com/nais/digdirator/pkg/metrics"
	"github.com/nais/digdirator/pkg/secrets"
	log "github.com/sirupsen/logrus"
	"gopkg.in/square/go-jose.v2"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/tools/record"
	"net/http"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"time"
)

const RequeueInterval = 10 * time.Second

type Reconciler struct {
	Client client.Client
	Reader client.Reader
	Scheme *runtime.Scheme

	Recorder   record.EventRecorder
	Config     *config.Config
	Signer     jose.Signer
	HttpClient *http.Client
}

func NewReconciler(
	client client.Client,
	reader client.Reader,
	scheme *runtime.Scheme,
	recorder record.EventRecorder,
	config *config.Config,
	signer jose.Signer,
	httpClient *http.Client,
) Reconciler {
	return Reconciler{
		Client:     client,
		Reader:     reader,
		Scheme:     scheme,
		Recorder:   recorder,
		Config:     config,
		Signer:     signer,
		HttpClient: httpClient,
	}
}

func (r *Reconciler) Reconcile(req ctrl.Request, instance v1.Instance) (ctrl.Result, error) {
	tx, err := r.prepare(req, instance)
	if err != nil {
		return ctrl.Result{}, client.IgnoreNotFound(err)
	}

	defer func() {
		tx.Logger.Infof("finished processing request")
	}()

	finalizer := r.finalizer(tx)

	if tx.Instance.IsBeingDeleted() {
		return finalizer.Process()
	}

	if !tx.Instance.HasFinalizer(FinalizerName) {
		return finalizer.Register()
	}

	if hashUnchanged, err := tx.Instance.IsHashUnchanged(); hashUnchanged {
		if err != nil {
			return ctrl.Result{}, err
		}
		tx.Logger.Info("object state already reconciled, nothing to do")
		return ctrl.Result{}, nil
	}

	if err := r.process(tx); err != nil {
		return r.handleError(tx, err)
	}

	return r.complete(tx)
}

func (r *Reconciler) prepare(req ctrl.Request, instance v1.Instance) (*Transaction, error) {
	instanceType := instance.GetInstanceType()
	correlationID := uuid.New().String()

	logger := *log.WithFields(log.Fields{
		instanceType:    req.NamespacedName,
		"correlationID": correlationID,
	})

	ctx := context.Background()

	if err := r.Reader.Get(ctx, req.NamespacedName, instance); err != nil {
		return nil, err
	}
	instance.SetClusterName(r.Config.ClusterName)
	instance.GetStatus().SetCorrelationID(correlationID)

	digdirClient := digdir.NewClient(r.HttpClient, r.Signer, r.Config)

	logger.Infof("processing %s...", instanceType)

	transaction := NewTransaction(
		ctx,
		instance,
		&logger,
		&digdirClient,
	)
	return &transaction, nil
}

func (r *Reconciler) process(tx *Transaction) error {
	instanceClient, err := tx.DigdirClient.ClientExists(tx.Instance, tx.Ctx)
	if err != nil {
		return fmt.Errorf("checking if client exists: %w", err)
	}

	registrationPayload := tx.Instance.ToClientRegistration()
	if instanceClient != nil {
		instanceClient, err = r.updateClient(tx, registrationPayload, instanceClient.ClientID)
		metrics.IncClientsUpdated(tx.Instance)
	} else {
		instanceClient, err = r.createClient(tx, registrationPayload)
		metrics.IncClientsCreated(tx.Instance)
	}
	if err != nil {
		return err
	}

	if len(tx.Instance.GetStatus().GetClientID()) == 0 {
		tx.Instance.GetStatus().SetClientID(instanceClient.ClientID)
	}

	jwk, err := crypto.GenerateJwk()
	if err != nil {
		return fmt.Errorf("generating new JWK for client: %w", err)
	}

	secretsClient := r.secrets(tx)
	managedSecrets, err := secretsClient.GetManaged()
	if err != nil {
		return fmt.Errorf("getting managed secrets: %w", err)
	}

	if err := r.registerJwk(tx, *jwk, *managedSecrets, instanceClient.ClientID); err != nil {
		return err
	}
	metrics.IncClientsRotated(tx.Instance)

	if err := secretsClient.CreateOrUpdate(*jwk); err != nil {
		return fmt.Errorf("creating or updating secret: %w", err)
	}

	if err := secretsClient.DeleteUnused(managedSecrets.Unused); err != nil {
		return err
	}

	return nil
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

func (r *Reconciler) registerJwk(tx *Transaction, jwk jose.JSONWebKey, managedSecrets secrets.Lists, clientID string) error {
	jwks, err := crypto.MergeJwks(jwk, managedSecrets.Used, tx.Instance.GetSecretMapKey())
	if err != nil {
		return fmt.Errorf("merging new JWK with JWKs in use: %w", err)
	}

	tx.Logger.Debug("generated new JWKS for client, registering...")

	jwksResponse, err := tx.DigdirClient.RegisterKeys(tx.Ctx, clientID, jwks)

	if err != nil {
		return fmt.Errorf("registering JWKS for client at Digdir: %w", err)
	}

	tx.Instance.GetStatus().SetKeyIDs(crypto.KeyIDsFromJwks(&jwksResponse.JSONWebKeySet))
	tx.Logger = tx.Logger.WithField("KeyIDs", tx.Instance.GetStatus().GetKeyIDs())
	tx.Logger.Info("new JWKS for client registered")

	return nil
}

func (r *Reconciler) handleError(tx *Transaction, err error) (ctrl.Result, error) {
	tx.Logger.Error(fmt.Errorf("processing client: %w", err))
	r.Recorder.Event(tx.Instance, corev1.EventTypeWarning, "Failed", fmt.Sprintf("Failed to synchronize client, retrying in %s", RequeueInterval))
	metrics.IncClientsFailedProcessing(tx.Instance)

	return ctrl.Result{RequeueAfter: RequeueInterval}, nil
}

func (r *Reconciler) complete(tx *Transaction) (ctrl.Result, error) {
	tx.Logger.Debugf("updating status for %s", tx.Instance.GetInstanceType())

	hash, err := tx.Instance.CalculateHash()
	if err != nil {
		return ctrl.Result{}, err
	}
	tx.Instance.GetStatus().SetHash(hash)

	if err := r.Client.Status().Update(tx.Ctx, tx.Instance); err != nil {
		return ctrl.Result{}, fmt.Errorf("updating status subresource: %w", err)
	}

	tx.Logger.Info("status subresource successfully updated")

	r.Recorder.Event(tx.Instance, corev1.EventTypeNormal, "Synchronized", "client is up-to-date")
	tx.Logger.Info("successfully reconciled")
	metrics.IncClientsProcessed(tx.Instance)

	return ctrl.Result{}, nil
}
