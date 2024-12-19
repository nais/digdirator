package common

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/go-jose/go-jose/v4"
	naisiov1 "github.com/nais/liberator/pkg/apis/nais.io/v1"
	"github.com/nais/liberator/pkg/kubernetes"
	log "github.com/sirupsen/logrus"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/tools/record"
	"k8s.io/utils/ptr"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"

	"github.com/nais/digdirator/pkg/clients"
	"github.com/nais/digdirator/pkg/config"
	"github.com/nais/digdirator/pkg/crypto"
	"github.com/nais/digdirator/pkg/digdir"
	"github.com/nais/digdirator/pkg/digdir/types"
	"github.com/nais/digdirator/pkg/metrics"
)

type Reconciler struct {
	Client       client.Client
	Reader       client.Reader
	Scheme       *runtime.Scheme
	Recorder     record.EventRecorder
	Config       *config.Config
	DigDirClient digdir.Client
}

func NewReconciler(
	client client.Client,
	reader client.Reader,
	scheme *runtime.Scheme,
	recorder record.EventRecorder,
	config *config.Config,
	digdirClient digdir.Client,
) Reconciler {
	return Reconciler{
		Client:       client,
		Reader:       reader,
		Scheme:       scheme,
		Recorder:     recorder,
		Config:       config,
		DigDirClient: digdirClient,
	}
}

func (r *Reconciler) Reconcile(ctx context.Context, req ctrl.Request, instance clients.Instance) (ctrl.Result, error) {
	tx, err := r.prepare(ctx, req, instance)
	if err != nil {
		return ctrl.Result{}, client.IgnoreNotFound(err)
	}

	if markedForDeletion(tx.Instance) {
		return r.finalize(tx)
	}

	if !controllerutil.ContainsFinalizer(tx.Instance, FinalizerName) {
		controllerutil.AddFinalizer(tx.Instance, FinalizerName)
		if err := r.Client.Update(tx.Ctx, tx.Instance); err != nil {
			return ctrl.Result{}, fmt.Errorf("registering finalizer: %w", err)
		}
	}

	if clients.IsUpToDate(tx.Instance) && !HasRetryableStatusCondition(tx.Instance.GetStatus().Conditions) {
		tx.Logger.Info("resource is up-to-date; skipping reconciliation")
		// requeue later to re-evaluate and prevent resource drift
		return ctrl.Result{RequeueAfter: 8 * time.Hour}, nil
	}

	if err = r.process(tx); err != nil {
		if err := r.observeError(tx, err); err != nil {
			return ctrl.Result{}, fmt.Errorf("observing error: %w", err)
		}

		return ctrl.Result{Requeue: true}, nil
	}

	conditions := tx.Instance.GetStatus().Conditions
	switch {
	case IsStatusConditionTrue(conditions, ConditionTypeInvalidConsumedScopes):
		requeueAfter := 1 * time.Hour
		tx.Logger.Infof("resource has invalid consumed scopes; requeuing reconciliation after %s", requeueAfter)
		return ctrl.Result{RequeueAfter: requeueAfter}, nil

	case IsStatusConditionTrue(conditions, ConditionTypeInvalidExposedScopesConsumers):
		tx.Logger.Info("resource has invalid consumers in exposed scopes; will not requeue reconciliation")
		return ctrl.Result{}, nil

	default:
		tx.Logger.Info("successfully reconciled")
		metrics.IncClientsProcessed(tx.Instance)
		// requeue immediately to re-evaluate resource
		return ctrl.Result{Requeue: true}, nil
	}
}

func (r *Reconciler) prepare(ctx context.Context, req ctrl.Request, instance clients.Instance) (*Transaction, error) {
	instanceType := clients.GetInstanceType(instance)
	correlationID := controller.ReconcileIDFromContext(ctx)

	if err := r.Reader.Get(ctx, req.NamespacedName, instance); err != nil {
		return nil, err
	}

	status := instance.GetStatus()

	fields := log.Fields{
		"instance_type":      instanceType,
		"instance_name":      req.Name,
		"instance_namespace": req.Namespace,
		"correlation_id":     correlationID,
		"client_id":          status.ClientID,
		"key_ids":            strings.Join(status.KeyIDs, ", "),
	}

	return NewTransaction(ctx, instance, log.WithFields(fields)), nil
}

func (r *Reconciler) process(tx *Transaction) error {
	status := tx.Instance.GetStatus()
	status.SetCondition(ReadyCondition(
		metav1.ConditionFalse,
		ConditionReasonProcessing,
		"Started processing resource",
		tx.Instance.GetGeneration()),
	)
	if err := r.Client.Status().Update(tx.Ctx, tx.Instance); err != nil {
		return fmt.Errorf("updating status: %w", err)
	}

	registration, err := r.createOrUpdateClient(tx)
	if err != nil {
		return err
	}

	status.ClientID = registration.ClientID

	secretsClient := r.secrets(tx)
	managedSecrets, err := secretsClient.GetManaged()
	if err != nil {
		return fmt.Errorf("getting managed secrets: %w", err)
	}

	var jwk *jose.JSONWebKey

	if clients.NeedsSecretRotation(tx.Instance) {
		jwk, err = crypto.GenerateJwk()
		if err != nil {
			return fmt.Errorf("generating jwk: %w", err)
		}

		if err := r.registerJwk(tx, *jwk, managedSecrets, registration.ClientID); err != nil {
			return err
		}

		r.reportEvent(tx, corev1.EventTypeNormal, EventRotatedInDigDir, "Client credentials is rotated")
		metrics.IncClientsRotated(tx.Instance)
	} else {
		jwk, err = crypto.GetPreviousJwkFromSecret(managedSecrets, clients.GetSecretJwkKey(tx.Instance))
		if err != nil {
			if errors.Is(err, crypto.ErrNoPreviousJwkFound) {
				tx.Logger.Warn("no previous JWK found in secrets, generating one...")
				jwk, err = crypto.GenerateJwk()
				if err != nil {
					return fmt.Errorf("generating new JWK: %w", err)
				}
			} else {
				return err
			}
		}

		if err := r.ensureJwkValidExternalState(tx, registration, jwk, managedSecrets); err != nil {
			return fmt.Errorf("refreshing keys: %w", err)
		}
	}

	if err := secretsClient.CreateOrUpdate(*jwk); err != nil {
		return fmt.Errorf("creating or updating secret: %w", err)
	}

	if err := secretsClient.DeleteUnused(managedSecrets.Unused); err != nil {
		return err
	}

	// object is overwritten with response from apiserver after Update, so status is unset
	// preserve copy for update of status subresource later on
	status = tx.Instance.GetStatus().DeepCopy()

	// remove processed annotations
	a := tx.Instance.GetAnnotations()
	_, hasResync := a[clients.AnnotationResynchronize]
	_, hasRotate := a[clients.AnnotationRotate]

	if hasResync || hasRotate {
		delete(a, clients.AnnotationResynchronize)
		delete(a, clients.AnnotationRotate)

		if err := r.Client.Update(tx.Ctx, tx.Instance); err != nil {
			return fmt.Errorf("updating object: %w", err)
		}
	}

	// update status
	hash, err := tx.Instance.Hash()
	if err != nil {
		return err
	}
	generation := tx.Instance.GetGeneration()
	status.CorrelationID = string(controller.ReconcileIDFromContext(tx.Ctx))
	status.ObservedGeneration = ptr.To(generation)
	status.SynchronizationHash = hash
	status.SynchronizationSecretName = clients.GetSecretName(tx.Instance)
	status.SetStateSynchronized()
	status.SetCondition(ReadyCondition(
		metav1.ConditionTrue,
		ConditionReasonSynchronized,
		"Resource is up-to-date with DigDir",
		generation),
	)
	status.SetCondition(ErrorCondition(
		metav1.ConditionFalse,
		ConditionReasonSynchronized,
		"Processing completed without errors",
		generation),
	)

	r.reportEvent(tx, corev1.EventTypeNormal, EventSynchronized, "Resource is up-to-date")

	// re-apply status to object
	tx.Instance.SetStatus(*status)

	// finally, set status subresource
	if err := r.Client.Status().Update(tx.Ctx, tx.Instance); err != nil {
		return fmt.Errorf("updating status: %w", err)
	}
	return nil
}

func (r *Reconciler) observeError(tx *Transaction, reconcileErr error) error {
	setStatusCondition := func(message string) {
		r.reportEvent(tx, corev1.EventTypeWarning, EventFailedSynchronization, message)
		tx.Instance.GetStatus().SetCondition(ErrorCondition(
			metav1.ConditionTrue,
			ConditionReasonFailed,
			message,
			tx.Instance.GetGeneration()),
		)
	}

	var digdirErr *digdir.Error
	if errors.As(reconcileErr, &digdirErr) {
		setStatusCondition(digdirErr.Message)
	} else {
		setStatusCondition(reconcileErr.Error())
	}

	tx.Logger.Errorf("processing resource: %+v", reconcileErr)
	metrics.IncClientsFailedProcessing(tx.Instance)
	return r.Client.Status().Update(tx.Ctx, tx.Instance)
}

func (r *Reconciler) createOrUpdateClient(tx *Transaction) (*types.ClientRegistration, error) {
	registration, err := r.DigDirClient.GetRegistration(tx.Instance, tx.Ctx, r.Config.ClusterName)
	if err != nil {
		return nil, fmt.Errorf("getting client registration: %w", err)
	}

	registrationPayload := clients.ToClientRegistration(tx.Instance, r.Config)

	switch instance := tx.Instance.(type) {
	case *naisiov1.MaskinportenClient:
		exposedScopes := instance.GetExposedScopes()
		scopes := r.scopes(tx)

		err := scopes.Process(exposedScopes)
		if err != nil {
			return nil, fmt.Errorf("processing scopes: %w", err)
		}

		consumedScopes, err := r.filterConsumedScopes(tx, instance)
		if err != nil {
			return nil, err
		}

		registrationPayload.Scopes = consumedScopes
		tx.Logger.Infof("registering client scopes: [%s]", strings.Join(consumedScopes, ", "))
	}

	if registration != nil {
		existingType := registration.IntegrationType
		desiredType := registrationPayload.IntegrationType

		if existingType != desiredType {
			return nil, fmt.Errorf("cannot update immutable integration type (existing: %s, desired: %s)", existingType, desiredType)
		}

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

	registrationResponse, err := r.DigDirClient.Register(tx.Ctx, payload)
	if err != nil {
		return nil, fmt.Errorf("registering client: %w", err)
	}

	tx.Logger = tx.Logger.WithField("client_id", registrationResponse.ClientID)
	tx.Logger.Info("client registered")
	return registrationResponse, nil
}

func (r *Reconciler) updateClient(tx *Transaction, payload types.ClientRegistration, clientID string) (*types.ClientRegistration, error) {
	tx.Logger = tx.Logger.WithField("client_id", clientID)
	tx.Logger.Debug("client already exists, updating...")

	registrationResponse, err := r.DigDirClient.Update(tx.Ctx, payload, clientID)
	if err != nil {
		return nil, fmt.Errorf("updating client: %w", err)
	}

	tx.Logger.Info("client updated")
	return registrationResponse, err
}

func (r *Reconciler) filterConsumedScopes(tx *Transaction, client *naisiov1.MaskinportenClient) ([]string, error) {
	desired := client.Spec.Scopes.ConsumedScopes

	// hack: set default scopes if none are specified, because we cannot register a client without scopes
	// this is only relevant for MaskinportenClients that _only_ expose scopes.
	if len(desired) == 0 {
		desired = []naisiov1.ConsumedScope{{Name: r.Config.DigDir.Maskinporten.Default.ClientScope}}
	}

	valid := make([]string, 0)
	invalid := make([]string, 0)
	for _, scp := range desired {
		canAccess, err := r.DigDirClient.CanAccessScope(tx.Ctx, scp)
		if err != nil {
			return nil, err
		}

		if canAccess {
			valid = append(valid, scp.Name)
			continue
		}

		invalid = append(invalid, scp.Name)
	}

	if len(invalid) > 0 {
		message := fmt.Sprintf("Organization has no access to scopes: [%s]", strings.Join(invalid, ", "))
		tx.Logger.Warn(message)
		tx.Instance.GetStatus().SetCondition(InvalidConsumedScopesCondition(
			metav1.ConditionTrue,
			ConditionReasonFailed,
			message,
			tx.Instance.GetGeneration()),
		)
	} else {
		tx.Instance.GetStatus().SetCondition(InvalidConsumedScopesCondition(
			metav1.ConditionFalse,
			ConditionReasonValidated,
			"Organization has access to all consumed scopes",
			tx.Instance.GetGeneration()),
		)
	}

	return valid, nil
}

func (r *Reconciler) registerJwk(tx *Transaction, jwk jose.JSONWebKey, managedSecrets kubernetes.SecretLists, clientID string) error {
	jwks, err := crypto.MergeJwks(jwk, managedSecrets.Used, clients.GetSecretJwkKey(tx.Instance))
	if err != nil {
		return fmt.Errorf("merging JWKS: %w", err)
	}

	tx.Logger.Debug("generated new JWKS for client, registering...")

	jwksResponse, err := r.DigDirClient.RegisterKeys(tx.Ctx, clientID, jwks)
	if err != nil {
		return fmt.Errorf("registering JWKS: %w", err)
	}

	tx.Instance.GetStatus().KeyIDs = jwksResponse.KeyIDs()
	tx.Logger = tx.Logger.WithField("key_ids", strings.Join(tx.Instance.GetStatus().KeyIDs, ", "))
	tx.Logger.Info("new JWKS for client registered")

	return nil
}

// ensureJwkValidExternalState ensures that the JWK is registered in DigDir and is not expiring soon.
func (r *Reconciler) ensureJwkValidExternalState(tx *Transaction, registration *types.ClientRegistration, jwk *jose.JSONWebKey, managedSecrets kubernetes.SecretLists) error {
	resp, err := r.DigDirClient.GetKeys(tx.Ctx, registration.ClientID)
	if err != nil {
		return fmt.Errorf("getting keys: %w", err)
	}

	const expiryThreshold = 30 * 24 * time.Hour // 30 days

	found := false
	expiring := false
	for _, key := range resp.Keys {
		if key.KeyID == jwk.KeyID {
			found = true

			if time.Until(key.ExpiryTime()) < expiryThreshold {
				tx.Logger.Infof("key %q expires at %q, refreshing...", key.KeyID, key.ExpiryTime())
				expiring = true
			}

			break
		}
	}

	if found && !expiring {
		return nil
	}
	if !found {
		tx.Logger.Infof("key %q not found in DigDir, registering...", jwk.KeyID)
	}

	if err := r.registerJwk(tx, *jwk, managedSecrets, registration.ClientID); err != nil {
		return err
	}

	r.reportEvent(tx, corev1.EventTypeNormal, EventUpdatedInDigDir, "Client credentials is refreshed")
	return nil
}

func (r *Reconciler) reportEvent(tx *Transaction, eventType, event, message string) {
	status := tx.Instance.GetStatus()
	status.SynchronizationState = event
	r.Recorder.Event(tx.Instance, eventType, event, message)
}
