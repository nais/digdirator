package idportenclient

import (
	"context"
	"fmt"
	"github.com/nais/digdirator/pkg/config"
	"github.com/nais/digdirator/pkg/idporten"
	"time"

	"github.com/google/uuid"
	"github.com/nais/digdirator/api/v1"
	"github.com/nais/digdirator/pkg/secrets"
	log "github.com/sirupsen/logrus"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/tools/record"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

const requeueInterval = 10 * time.Second

// IDPortenClientReconciler reconciles a IDPortenClient object
type Reconciler struct {
	client.Client
	Reader client.Reader
	Scheme *runtime.Scheme

	Recorder       record.EventRecorder
	Config         *config.Config
	IDPortenClient idporten.Client
}

type transaction struct {
	ctx      context.Context
	instance *v1.IDPortenClient
	log      log.Entry
}

// +kubebuilder:rbac:groups=nais.io,resources=IDPortenClients,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=nais.io,resources=IDPortenClients/status,verbs=get;update;patch;create
// +kubebuilder:rbac:groups=*,resources=events,verbs=get;list;watch;create;update

func (r *Reconciler) Reconcile(req ctrl.Request) (ctrl.Result, error) {
	correlationId := uuid.New().String()

	logger := *log.WithFields(log.Fields{
		"IDPortenClient": req.NamespacedName,
		"correlationId":  correlationId,
	})

	tx, err := r.prepare(req, correlationId, logger)
	if err != nil {
		return ctrl.Result{}, client.IgnoreNotFound(err)
	}

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
		logger.Info("object state already reconciled, nothing to do")
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

func (r *Reconciler) prepare(req ctrl.Request, correlationId string, logger log.Entry) (transaction, error) {
	ctx := context.Background()

	instance := &v1.IDPortenClient{}
	if err := r.Reader.Get(ctx, req.NamespacedName, instance); err != nil {
		return transaction{}, err
	}
	instance.SetClusterName(r.Config.ClusterName)
	instance.Status.CorrelationId = correlationId
	logger.Info("processing IDPortenClient...")
	return transaction{ctx, instance, logger}, nil
}

func (r *Reconciler) process(tx transaction) error {
	managedSecrets, err := secrets.GetManaged(tx.ctx, tx.instance, r.Reader)
	if err != nil {
		return err
	}

	client, err := r.IDPortenClient.ClientExists(tx.instance.ClientID(), tx.ctx)
	if err != nil {
		return fmt.Errorf("checking if client exists: %w", err)
	}
	if client != nil {
		// update
		tx.log.Info(client)
	} else {

		// create
	}

	// todo - register/update idporten client
	// todo - generate and register JWKS
	// 	- JWKS should contain in-use JWK (in-use => mounted to an existing pod) and newly generated JWK
	// 	- idporten only accepts max 5 JWKs in JWKS - should check if pod status is Running

	if err := r.secrets().createOrUpdate(tx); err != nil {
		return err
	}

	if err := r.secrets().deleteUnused(tx, managedSecrets.Unused); err != nil {
		return err
	}

	return nil
}

func (r *Reconciler) handleError(tx transaction, err error) (ctrl.Result, error) {
	tx.log.Error(fmt.Errorf("failed to process ID-porten client: %w", err))
	r.Recorder.Event(tx.instance, corev1.EventTypeWarning, "Failed", fmt.Sprintf("Failed to synchronize ID-porten client, retrying in %s", requeueInterval))

	return ctrl.Result{RequeueAfter: requeueInterval}, nil
}

func (r *Reconciler) complete(tx transaction) (ctrl.Result, error) {
	if err := r.updateStatus(tx); err != nil {
		return ctrl.Result{}, err
	}

	r.Recorder.Event(tx.instance, corev1.EventTypeNormal, "Synchronized", "ID-porten client is up-to-date")
	tx.log.Info("successfully reconciled")

	return ctrl.Result{}, nil
}

func (r *Reconciler) updateStatus(tx transaction) error {
	tx.log.Debug("updating status for IDPortenClient")
	tx.instance.Status.ClientId = "todo"

	if err := tx.instance.UpdateHash(); err != nil {
		return err
	}
	if err := r.Status().Update(tx.ctx, tx.instance); err != nil {
		return fmt.Errorf("failed to update status subresource: %w", err)
	}

	tx.log.WithFields(
		log.Fields{
			"ClientID": tx.instance.Status.ClientId,
		}).Info("status subresource successfully updated")
	return nil
}
