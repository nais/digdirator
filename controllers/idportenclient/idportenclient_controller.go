package idportenclient

import (
	"context"
	"fmt"
	"github.com/google/uuid"
	v1 "github.com/nais/digdirator/api/v1"
	"github.com/nais/digdirator/controllers/common"
	"github.com/nais/digdirator/controllers/common/reconciler"
	transaction2 "github.com/nais/digdirator/controllers/common/transaction"
	"github.com/nais/digdirator/controllers/finalizer"
	"github.com/nais/digdirator/pkg/crypto"
	"github.com/nais/digdirator/pkg/digdir"
	"github.com/nais/digdirator/pkg/digdir/types"
	"github.com/nais/digdirator/pkg/metrics"
	"github.com/nais/digdirator/pkg/secrets"
	log "github.com/sirupsen/logrus"
	"gopkg.in/square/go-jose.v2"
	corev1 "k8s.io/api/core/v1"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

type Reconciler struct {
	reconciler.Reconciler
}

type transaction struct {
	instance *v1.IDPortenClient
	transaction2.Transaction
}

// +kubebuilder:rbac:groups=nais.io,resources=IDPortenClients,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=nais.io,resources=IDPortenClients/status,verbs=get;update;patch;create
// +kubebuilder:rbac:groups=*,resources=events,verbs=get;list;watch;create;update

func (r *Reconciler) Reconcile(req ctrl.Request) (ctrl.Result, error) {
	tx, err := r.prepare(req)
	if err != nil {
		return ctrl.Result{}, client.IgnoreNotFound(err)
	}

	defer func() {
		tx.Logger.Infof("finished processing request")
	}()

	finalizerClient := finalizer.Client(r.Reconciler, tx.Transaction)

	if !common.HasFinalizer(tx.instance, finalizer.FinalizerName) {
		return finalizerClient.Register()
	}

	if common.InstanceIsBeingDeleted(tx.instance) {
		return finalizerClient.Process()
	}

	if hashUnchanged, err := tx.instance.HashUnchanged(); hashUnchanged {
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

func (r *Reconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&v1.IDPortenClient{}).
		Complete(r)
}

func (r *Reconciler) prepare(req ctrl.Request) (*transaction, error) {
	correlationID := uuid.New().String()

	logger := *log.WithFields(log.Fields{
		"IDPortenClient": req.NamespacedName,
		"correlationID":  correlationID,
	})

	ctx := context.Background()

	instance := &v1.IDPortenClient{}
	if err := r.Reader.Get(ctx, req.NamespacedName, instance); err != nil {
		return nil, err
	}
	instance.SetClusterName(r.Config.ClusterName)
	instance.Status.CorrelationID = correlationID

	secretsClient := secrets.NewClient(ctx, instance, &logger, r.Reconciler)

	managedSecrets, err := secretsClient.GetManaged()
	if err != nil {
		return nil, fmt.Errorf("getting managed secrets: %w", err)
	}

	idportenClient := digdir.NewClient(r.HttpClient, r.Signer, r.Config)

	logger.Info("processing IDPortenClient...")

	return &transaction{Transaction: transaction2.Transaction{
		Ctx:            ctx,
		Logger:         &logger,
		ManagedSecrets: managedSecrets,
		DigdirClient:   &idportenClient,
		SecretsClient:  secretsClient,
		Instance:       instance,
	}, instance: instance}, nil
}

func (r *Reconciler) process(tx *transaction) error {
	idportenClient, err := tx.DigdirClient.ClientExists(tx.instance, tx.Ctx)
	if err != nil {
		return fmt.Errorf("checking if client exists: %w", err)
	}

	registrationPayload := tx.instance.ToClientRegistration()
	if idportenClient != nil {
		idportenClient, err = r.updateClient(tx, registrationPayload, idportenClient.ClientID)
		metrics.IncWithNamespaceLabel(metrics.IDPortenClientsUpdatedCount, tx.instance.GetNamespace())
	} else {
		idportenClient, err = r.createClient(tx, registrationPayload)
		metrics.IncWithNamespaceLabel(metrics.IDPortenClientsCreatedCount, tx.instance.GetNamespace())
	}
	if err != nil {
		return err
	}

	tx.Logger = tx.Logger.WithField("ClientID", idportenClient.ClientID)
	if len(tx.instance.Status.ClientID) == 0 {
		tx.instance.Status.ClientID = idportenClient.ClientID
	}

	jwk, err := crypto.GenerateJwk()
	if err != nil {
		return fmt.Errorf("generating new JWK for client: %w", err)
	}

	if err := r.registerJwk(tx, *jwk, idportenClient.ClientID); err != nil {
		return err
	}
	metrics.IncWithNamespaceLabel(metrics.IDPortenClientsRotatedCount, tx.instance.GetNamespace())

	if err := tx.SecretsClient.CreateOrUpdate(*jwk); err != nil {
		return fmt.Errorf("creating or updating secret: %w", err)
	}

	if err := tx.SecretsClient.DeleteUnused(tx.ManagedSecrets.Unused); err != nil {
		return err
	}

	return nil
}

func (r *Reconciler) createClient(tx *transaction, payload types.ClientRegistration) (*types.ClientRegistration, error) {
	tx.Logger.Debug("client does not exist in ID-porten, registering...")

	idportenClient, err := tx.DigdirClient.Register(tx.Ctx, payload)
	if err != nil {
		return nil, fmt.Errorf("registering client to ID-porten: %w", err)
	}

	tx.Logger.WithField("ClientID", idportenClient.ClientID).Info("client registered")
	return idportenClient, nil
}

func (r *Reconciler) updateClient(tx *transaction, payload types.ClientRegistration, clientID string) (*types.ClientRegistration, error) {
	tx.Logger.Debug("client already exists in ID-porten, updating...")

	idportenClient, err := tx.DigdirClient.Update(tx.Ctx, payload, clientID)
	if err != nil {
		return nil, fmt.Errorf("updating client at ID-porten: %w", err)
	}

	tx.Logger.WithField("ClientID", idportenClient.ClientID).Info("client updated")
	return idportenClient, err
}

func (r *Reconciler) registerJwk(tx *transaction, jwk jose.JSONWebKey, clientID string) error {
	jwks, err := crypto.MergeJwks(jwk, tx.ManagedSecrets.Used, secrets.IDPortenJwkKey)
	if err != nil {
		return fmt.Errorf("merging new JWK with JWKs in use: %w", err)
	}

	tx.Logger.Debug("generated new JWKS for client, registering...")

	jwksResponse, err := tx.DigdirClient.RegisterKeys(tx.Ctx, clientID, jwks)

	if err != nil {
		return fmt.Errorf("registering JWKS for client at ID-porten: %w", err)
	}

	tx.instance.Status.KeyIDs = crypto.KeyIDsFromJwks(&jwksResponse.JSONWebKeySet)
	tx.Logger = tx.Logger.WithField("KeyIDs", tx.instance.Status.KeyIDs)
	tx.Logger.Info("new JWKS for client registered")

	return nil
}

func (r *Reconciler) handleError(tx *transaction, err error) (ctrl.Result, error) {
	tx.Logger.Error(fmt.Errorf("processing ID-porten client: %w", err))
	r.Recorder.Event(tx.instance, corev1.EventTypeWarning, "Failed", fmt.Sprintf("Failed to synchronize ID-porten client, retrying in %s", reconciler.RequeueInterval))
	metrics.IncWithNamespaceLabel(metrics.IDPortenClientsFailedProcessingCount, tx.instance.GetNamespace())
	return ctrl.Result{RequeueAfter: reconciler.RequeueInterval}, nil
}

func (r *Reconciler) complete(tx *transaction) (ctrl.Result, error) {
	tx.Logger.Debug("updating status for IDPortenClient")

	if err := tx.instance.UpdateHash(); err != nil {
		return ctrl.Result{}, err
	}
	if err := r.Client.Status().Update(tx.Ctx, tx.instance); err != nil {
		return ctrl.Result{}, fmt.Errorf("updating status subresource: %w", err)
	}

	tx.Logger.Info("status subresource successfully updated")

	r.Recorder.Event(tx.instance, corev1.EventTypeNormal, "Synchronized", "ID-porten client is up-to-date")

	tx.Logger.Info("successfully reconciled")

	metrics.IncWithNamespaceLabel(metrics.IDPortenClientsProcessedCount, tx.instance.GetNamespace())

	return ctrl.Result{}, nil
}
