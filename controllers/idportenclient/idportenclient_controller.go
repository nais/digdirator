package idportenclient

import (
	v1 "github.com/nais/digdirator/api/v1"
	"github.com/nais/digdirator/controllers/common"
	ctrl "sigs.k8s.io/controller-runtime"
)

type IDPortenReconciler struct {
	common.Reconciler
}

func NewReconciler(reconciler common.Reconciler) *IDPortenReconciler {
	return &IDPortenReconciler{Reconciler: reconciler}
}

// +kubebuilder:rbac:groups=nais.io,resources=IDPortenClients,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=nais.io,resources=IDPortenClients/status,verbs=get;update;patch;create
// +kubebuilder:rbac:groups=*,resources=events,verbs=get;list;watch;create;update

func (r *IDPortenReconciler) Reconcile(req ctrl.Request) (ctrl.Result, error) {
	return r.Reconciler.Reconcile(req, &v1.IDPortenClient{})
}

func (r *IDPortenReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&v1.IDPortenClient{}).
		Complete(r)
}
