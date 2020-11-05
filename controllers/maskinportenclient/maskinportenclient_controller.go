package maskinportenclient

import (
	"context"
	"fmt"
	"github.com/nais/digdirator/controllers"
	"github.com/nais/digdirator/pkg/config"
	"github.com/nais/digdirator/pkg/crypto"
	"github.com/nais/digdirator/pkg/digdir"
	"github.com/nais/digdirator/pkg/digdir/types"
	"github.com/nais/digdirator/pkg/metrics"
	"gopkg.in/square/go-jose.v2"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/tools/record"
	"net/http"
	"time"

	"github.com/google/uuid"
	v1 "github.com/nais/digdirator/api/v1"
	"github.com/nais/digdirator/pkg/secrets"
	log "github.com/sirupsen/logrus"
	corev1 "k8s.io/api/core/v1"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

const requeueInterval = 10 * time.Second

// ClientReconciler reconciles a Client object
type Reconciler struct {
	Client client.Client
	Reader client.Reader
	Scheme *runtime.Scheme

	Recorder   record.EventRecorder
	Config     *config.Config
	Signer     jose.Signer
	HttpClient *http.Client
}

type transaction struct {
	ctx            context.Context
	instance       *v1.MaskinportenClient
	log            *log.Entry
	managedSecrets *secrets.Lists
	digdirClient   *digdir.Client
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
		tx.log.Infof("finished processing request")
	}()

	if tx.instance.IsBeingDeleted() {
		return r.finalizer().process(tx)
	}

	if !tx.instance.HasFinalizer(FinalizerName) {
		return r.finalizer().register(tx)
	}

	if hashUnchanged, err := tx.instance.HashUnchanged(); hashUnchanged {
		if err != nil {
			return ctrl.Result{}, err
		}
		tx.log.Info("object state already reconciled, nothing to do")
		return ctrl.Result{}, nil
	}

	if err := r.process(tx); err != nil {
		return r.handleError(tx, err)
	}

	return r.complete(tx)
}

func (r *Reconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&v1.MaskinportenClient{}).
		Complete(r)
}

func (r *Reconciler) prepare(req ctrl.Request) (*transaction, error) {
	correlationID := uuid.New().String()

	logger := *log.WithFields(log.Fields{
		"MaskinportenClient": req.NamespacedName,
		"correlationID":      correlationID,
	})

	ctx := context.Background()

	instance := &v1.MaskinportenClient{}
	instanceInterface := controllers.Instance(instance)
	if err := r.Reader.Get(ctx, req.NamespacedName, instance); err != nil {
		return nil, err
	}
	instance.SetClusterName(r.Config.ClusterName)
	instance.Status.CorrelationID = correlationID

	managedSecrets, err := secrets.GetManaged(ctx, instanceInterface, r.Reader)
	if err != nil {
		return nil, fmt.Errorf("getting managed secrets: %w", err)
	}

	digdirClient := digdir.NewClient(r.HttpClient, r.Signer, r.Config)

	logger.Info("processing MaskinportenClient...")

	return &transaction{
		ctx,
		instance,
		&logger,
		managedSecrets,
		&digdirClient,
	}, nil
}

func (r *Reconciler) process(tx *transaction) error {
	var instanceInterface = controllers.Instance(tx.instance)
	digdirClient, err := tx.digdirClient.ClientExists(instanceInterface, tx.ctx)
	if err != nil {
		return fmt.Errorf("checking if client exists: %w", err)
	}

	registrationPayload := tx.instance.ToClientRegistration()
	if digdirClient != nil {
		digdirClient, err = r.updateClient(tx, registrationPayload, digdirClient.ClientID)
		metrics.IncWithNamespaceLabel(metrics.IDPortenClientsUpdatedCount, tx.instance.Namespace)
	} else {
		digdirClient, err = r.createClient(tx, registrationPayload)
		metrics.IncWithNamespaceLabel(metrics.IDPortenClientsCreatedCount, tx.instance.Namespace)
	}
	if err != nil {
		return err
	}

	tx.log = tx.log.WithField("ClientID", digdirClient.ClientID)
	if len(tx.instance.Status.ClientID) == 0 {
		tx.instance.Status.ClientID = digdirClient.ClientID
	}

	jwk, err := crypto.GenerateJwk()
	if err != nil {
		return fmt.Errorf("generating new JWK for client: %w", err)
	}

	if err := r.registerJwk(tx, *jwk, digdirClient.ClientID); err != nil {
		return err
	}
	metrics.IncWithNamespaceLabel(metrics.IDPortenClientsRotatedCount, tx.instance.Namespace)

	if err := r.secrets().createOrUpdate(tx, *jwk); err != nil {
		return err
	}

	if err := r.secrets().deleteUnused(tx, tx.managedSecrets.Unused); err != nil {
		return err
	}

	return nil
}

func (r *Reconciler) createClient(tx *transaction, payload types.ClientRegistration) (*types.ClientRegistration, error) {
	tx.log.Debug("client does not exist in ID-porten, registering...")

	digdirClient, err := tx.digdirClient.Register(tx.ctx, payload)
	if err != nil {
		return nil, fmt.Errorf("registering client to ID-porten: %w", err)
	}

	tx.log.WithField("ClientID", digdirClient.ClientID).Info("client registered")
	return digdirClient, nil
}

func (r *Reconciler) updateClient(tx *transaction, payload types.ClientRegistration, clientID string) (*types.ClientRegistration, error) {
	tx.log.Debug("client already exists in ID-porten, updating...")

	digdirClient, err := tx.digdirClient.Update(tx.ctx, payload, clientID)
	if err != nil {
		return nil, fmt.Errorf("updating client at ID-porten: %w", err)
	}

	tx.log.WithField("ClientID", digdirClient.ClientID).Info("client updated")
	return digdirClient, err
}

func (r *Reconciler) registerJwk(tx *transaction, jwk jose.JSONWebKey, clientID string) error {
	jwks, err := crypto.MergeJwks(jwk, tx.managedSecrets.Used)
	if err != nil {
		return fmt.Errorf("merging new JWK with JWKs in use: %w", err)
	}

	tx.log.Debug("generated new JWKS for client, registering...")

	jwksResponse, err := tx.digdirClient.RegisterKeys(tx.ctx, clientID, jwks)

	if err != nil {
		return fmt.Errorf("registering JWKS for client at ID-porten: %w", err)
	}

	tx.instance.Status.KeyIDs = crypto.KeyIDsFromJwks(&jwksResponse.JSONWebKeySet)
	tx.log = tx.log.WithField("KeyIDs", tx.instance.Status.KeyIDs)
	tx.log.Info("new JWKS for client registered")

	return nil
}

func (r *Reconciler) handleError(tx *transaction, err error) (ctrl.Result, error) {
	tx.log.Error(fmt.Errorf("processing ID-porten client: %w", err))
	r.Recorder.Event(tx.instance, corev1.EventTypeWarning, "Failed", fmt.Sprintf("Failed to synchronize ID-porten client, retrying in %s", requeueInterval))
	metrics.IncWithNamespaceLabel(metrics.IDPortenClientsFailedProcessingCount, tx.instance.Namespace)
	return ctrl.Result{RequeueAfter: requeueInterval}, nil
}

func (r *Reconciler) complete(tx *transaction) (ctrl.Result, error) {
	tx.log.Debug("updating status for MaskinportenClient")

	if err := tx.instance.UpdateHash(); err != nil {
		return ctrl.Result{}, err
	}
	if err := r.Client.Status().Update(tx.ctx, tx.instance); err != nil {
		return ctrl.Result{}, fmt.Errorf("updating status subresource: %w", err)
	}

	tx.log.Info("status subresource successfully updated")

	r.Recorder.Event(tx.instance, corev1.EventTypeNormal, "Synchronized", "ID-porten client is up-to-date")

	tx.log.Info("successfully reconciled")

	metrics.IncWithNamespaceLabel(metrics.IDPortenClientsProcessedCount, tx.instance.Namespace)

	return ctrl.Result{}, nil
}
