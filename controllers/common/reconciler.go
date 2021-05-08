package common

import (
	"context"
	"fmt"
	"github.com/nais/digdirator/pkg/digdir/scopes"
	"net/http"
	"strconv"
	"sync"
	"time"

	"github.com/google/uuid"
	nais_io_v1 "github.com/nais/liberator/pkg/apis/nais.io/v1"
	v1 "github.com/nais/liberator/pkg/apis/nais.io/v1"
	finalizer2 "github.com/nais/liberator/pkg/finalizer"
	"github.com/nais/liberator/pkg/kubernetes"
	log "github.com/sirupsen/logrus"
	"gopkg.in/square/go-jose.v2"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/tools/record"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"

	"github.com/nais/digdirator/pkg/annotations"
	"github.com/nais/digdirator/pkg/clients"
	"github.com/nais/digdirator/pkg/config"
	"github.com/nais/digdirator/pkg/crypto"
	"github.com/nais/digdirator/pkg/digdir"
	"github.com/nais/digdirator/pkg/digdir/types"
	"github.com/nais/digdirator/pkg/metrics"
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

func (r *Reconciler) Reconcile(req ctrl.Request, instance clients.Instance) (ctrl.Result, error) {
	tx, err := r.prepare(req, instance)
	if err != nil {
		return ctrl.Result{}, client.IgnoreNotFound(err)
	}

	defer func() {
		tx.Logger.Infof("finished processing request")
	}()

	if r.shouldSkip(tx) {
		tx.Logger.Info("skipping processing of this resource")
		return ctrl.Result{}, nil
	}

	finalizer := r.finalizer(tx)

	if finalizer2.IsBeingDeleted(tx.Instance) {
		return finalizer.Process()
	}

	if !finalizer2.HasFinalizer(tx.Instance, FinalizerName) {
		return finalizer.Register()
	}

	inSharedNamespace, err := r.inSharedNamespace(tx)

	if err != nil {
		return ctrl.Result{}, err
	}

	if inSharedNamespace {
		if err := r.Client.Update(tx.Ctx, tx.Instance); err != nil {
			return ctrl.Result{}, fmt.Errorf("failed to update resource with skip flag: %w", err)
		}
		return ctrl.Result{}, nil
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

func (r *Reconciler) prepare(req ctrl.Request, instance clients.Instance) (*Transaction, error) {
	instanceType := clients.GetInstanceType(instance)
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

	digdirClient := digdir.NewClient(r.HttpClient, r.Signer, r.Config, instance)

	logger.Infof("processing %s...", instanceType)

	transaction := NewTransaction(
		ctx,
		instance,
		&logger,
		&digdirClient,
	)
	return &transaction, nil
}

func (r *Reconciler) shouldSkip(tx *Transaction) bool {
	if clients.HasSkipAnnotation(tx.Instance) {
		msg := fmt.Sprintf("Resource contains '%s' annotation. Skipping processing...", annotations.SkipKey)
		tx.Logger.Debug(msg)
		r.reportEvent(tx, corev1.EventTypeWarning, v1.EventSkipped, msg)
		return true
	}
	return false
}

func (r *Reconciler) inSharedNamespace(tx *Transaction) (bool, error) {
	sharedNs, err := kubernetes.ListSharedNamespaces(tx.Ctx, r.Reader)
	if err != nil {
		return false, err
	}
	for _, ns := range sharedNs.Items {
		if ns.Name == tx.Instance.GetNamespace() {
			msg := fmt.Sprintf("Resource should not exist in shared namespace '%s'. Skipping...", tx.Instance.GetNamespace())
			tx.Logger.Debug(msg)
			clients.SetAnnotation(tx.Instance, annotations.SkipKey, strconv.FormatBool(true))
			r.reportEvent(tx, corev1.EventTypeWarning, v1.EventNotInTeamNamespace, msg)
			r.reportEvent(tx, corev1.EventTypeWarning, v1.EventSkipped, msg)
			return true, nil
		}
	}
	return false, nil
}

func (r *Reconciler) process(tx *Transaction) error {
	instanceClient, err := tx.DigdirClient.ClientExists(tx.Instance, tx.Ctx)
	if err != nil {
		return fmt.Errorf("checking if client exists: %w", err)
	}

	registrationPayload := clients.ToClientRegistration(tx.Instance)

	switch tx.Instance.(type) {
	case *nais_io_v1.MaskinportenClient:
		filteredPayload, err := r.filterValidScopes(tx, registrationPayload)
		if err != nil {
			return err
		}
		registrationPayload = *filteredPayload
	}

	if instanceClient != nil {
		instanceClient, err = r.updateClient(tx, registrationPayload, instanceClient.ClientID)
		if err != nil {
			return err
		}

		r.reportEvent(tx, corev1.EventTypeNormal, EventUpdatedInDigDir, "Client is updated")
		metrics.IncClientsUpdated(tx.Instance)
	} else {
		instanceClient, err = r.createClient(tx, registrationPayload)
		if err != nil {
			return err
		}

		r.reportEvent(tx, corev1.EventTypeNormal, EventCreatedInDigDir, "Client is registered")
		metrics.IncClientsCreated(tx.Instance)
	}

	if len(tx.Instance.GetStatus().GetClientID()) == 0 {
		tx.Instance.GetStatus().SetClientID(instanceClient.ClientID)
	}

	if !clients.ShouldUpdateSecrets(tx.Instance) {
		return nil
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

	switch instance := tx.Instance.(type) {
	case *nais_io_v1.MaskinportenClient:
		actual, err := scopeExists(*tx, instance)
		if err != nil {
			return fmt.Errorf("checking if scopes exists: %w", err)
		}

		if actual != nil {
			if len(actual.CurrentScopes) > 0 {
				for _, scope := range actual.CurrentScopes {
					// update existing scopes and consumers
					scopeRegistration, consumerRegistration, err := r.UpdateScopeAndConsumers(tx, scope)
					if err != nil {
						return fmt.Errorf("updateing scopes and consumers: %w", err)
					}
					println(scopeRegistration)
					println(consumerRegistration)
				}
			}

			if len(actual.ScopeToCreate) > 0 {
				for _, newScope := range actual.ScopeToCreate {
					// create scopes and add consumers
					scope, consumers, err := r.CreateAndUpdateConsumers(tx, instance, newScope)
					if err != nil {
						return fmt.Errorf("creating scopes and adding consumers: %w", err)
					}
					println(scope)
					println(consumers)
				}
			}
		}
	}

	r.reportEvent(tx, corev1.EventTypeNormal, EventRotatedInDigDir, "Client credentials is rotated")
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

func (r *Reconciler) filterValidScopes(tx *Transaction, registration types.ClientRegistration) (*types.ClientRegistration, error) {
	var desiredScopes []v1.MaskinportenScope

	switch v := tx.Instance.(type) {
	case *nais_io_v1.IDPortenClient:
		return &registration, nil
	case *nais_io_v1.MaskinportenClient:
		desiredScopes = v.Spec.Scopes
	}

	accessibleScopes, err := tx.DigdirClient.GetAccessibleScopes(tx.Ctx)
	if err != nil {
		return nil, err
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
	tx.Logger = tx.Logger.WithField("KeyIDs", tx.Instance.GetStatus().GetKeyIDs())
	tx.Logger.Info("new JWKS for client registered")

	return nil
}

func (r *Reconciler) handleError(tx *Transaction, err error) (ctrl.Result, error) {
	tx.Logger.Error(fmt.Errorf("processing client: %w", err))
	r.reportEvent(tx, corev1.EventTypeWarning, EventFailedSynchronization, "Failed to synchronize client")

	metrics.IncClientsFailedProcessing(tx.Instance)
	r.reportEvent(tx, corev1.EventTypeNormal, EventRetrying, "Retrying synchronization")
	return ctrl.Result{RequeueAfter: RequeueInterval}, nil
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
		return r.Client.Update(tx.Ctx, existing)
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
		case *nais_io_v1.IDPortenClient:
			return &nais_io_v1.IDPortenClient{}
		case *nais_io_v1.MaskinportenClient:
			return &nais_io_v1.MaskinportenClient{}
		}
		return nil
	}(instance)

	err := r.Reader.Get(ctx, client.ObjectKey{Namespace: instance.GetNamespace(), Name: instance.GetName()}, existing)
	if err != nil {
		return fmt.Errorf("get newest version of %s: %s", clients.GetInstanceType(instance), err)
	}

	return updateFunc(existing)
}

func scopeExists(tx Transaction, instance *v1.MaskinportenClient) (*scopes.FilteredScopeContainer, error) {
	if instance.GetExternalScopes() != nil {
		scopeContainer, err := tx.DigdirClient.ScopesExists(tx.Instance, tx.Ctx, scopes.NewFilterForScope(instance.GetExternalScopes()))
		if err != nil {
			return nil, fmt.Errorf("getting list of scopes: %w", err)
		}
		return scopeContainer, nil
	}
	return nil, nil
}

func (r *Reconciler) createScope(tx *Transaction, payload types.ScopeRegistration) (*types.ScopeRegistration, error) {
	tx.Logger.Debug("scopes does not exist in Digdir, registering...")

	registrationResponse, err := tx.DigdirClient.RegisterScope(tx.Ctx, payload)
	if err != nil {
		return nil, fmt.Errorf("registering client to Digdir: %w", err)
	}

	tx.Logger = tx.Logger.WithField("scope", registrationResponse.Name)
	tx.Logger.Info("scope registered")
	return registrationResponse, nil
}

func (r *Reconciler) updateScope(tx *Transaction, container scopes.Scope) (*types.ScopeRegistration, error) {
	tx.Logger = tx.Logger.WithField("Subscope", container.ScopeRegistration.Name)
	tx.Logger.Debug("scope already exists in Digdir, updating...")

	registrationResponse, err := tx.DigdirClient.UpdateScope(tx.Ctx, container.ScopeRegistration, container.ScopeRegistration.Name)
	if err != nil {
		return nil, fmt.Errorf("updating scope at Digdir: %w", err)
	}

	tx.Logger.WithField("FilteredScopeContainer", registrationResponse.Name).Info("scope updated")
	return registrationResponse, err
}

func (r *Reconciler) updateConsumers(tx *Transaction, scope scopes.Scope) ([]types.ConsumerRegistration, error) {
	tx.Logger = tx.Logger.WithField("scope", scope.ToString())
	tx.Logger.Debug("Updating acl for scope...")

	acl, err := tx.DigdirClient.GetScopeACL(tx.Ctx, scope.ToString())
	if err != nil {
		return nil, fmt.Errorf("gettin ACL list from Digdir: %w", err)
	}

	consumerStatus, consumerList := scope.FilterConsumers(acl)
	registrationResponse := make([]types.ConsumerRegistration, 0)

	if len(consumerList) > 0 {
		for _, consumer := range consumerList {
			if consumer.Active {
				response, err := tx.DigdirClient.AddToConsumerACL(tx.Ctx, scope.ToString(), consumer.Orgno)
				if err != nil {
					return nil, fmt.Errorf("updating ACL list for scope at Digdir: %w", err)
				}
				consumerStatus = append(consumerStatus, consumer.Orgno)
				registrationResponse = append(registrationResponse, *response)
				tx.Logger.WithField("FilteredScopeContainer", response.Scope).Info("scope acl updated, added consumer(s)")
			} else {
				response, err := tx.DigdirClient.DeleteFromConsumerACL(tx.Ctx, scope.ToString(), consumer.Orgno)
				if err != nil {
					return nil, fmt.Errorf("updating ACL list for scope at Digdir: %w", err)
				}
				registrationResponse = append(registrationResponse, *response)
				tx.Logger.WithField("FilteredScopeContainer", response.Scope).Info("scope acl updated, deleted consumer(s)")
			}
		}
		tx.Instance.GetStatus().SetApplicationScopeConsumer(scope.ToString(), consumerStatus)
	}
	return registrationResponse, nil
}

func (r *Reconciler) UpdateScopeAndConsumers(tx *Transaction, scope scopes.Scope) (*types.ScopeRegistration, []types.ConsumerRegistration, error) {
	scopeRegistration, err := r.updateScope(tx, scope)
	if err != nil {
		return nil, nil, fmt.Errorf("updating scopes: %w", err)
	}
	consumerRegistration, err := r.updateConsumers(tx, scope)
	if err != nil {
		return nil, nil, fmt.Errorf("updating consumers: %w", err)
	}
	return scopeRegistration, consumerRegistration, nil
}

func (r *Reconciler) CreateAndUpdateConsumers(tx *Transaction, instance *v1.MaskinportenClient, newScope v1.ExternalScope) (*types.ScopeRegistration, []types.ConsumerRegistration, error) {
	scopeRegistrationPayload := clients.ToScopeRegistration(instance, newScope)
	scope, err := r.createScope(tx, scopeRegistrationPayload)
	if err != nil {
		return nil, nil, fmt.Errorf("creating scopes: %w", err)
	}

	scopeInfo := scopes.CreateScope(newScope.Consumers, scopeRegistrationPayload)
	consumers, err := r.updateConsumers(tx, scopeInfo)
	if err != nil {
		return nil, nil, fmt.Errorf("adding consumers: %w", err)
	}
	return scope, consumers, nil
}
