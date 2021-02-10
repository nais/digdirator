package maskinportenclient

import (
	"github.com/nais/digdirator/controllers/common"
	nais_io_v1 "github.com/nais/liberator/pkg/apis/nais.io/v1"
	ctrl "sigs.k8s.io/controller-runtime"
)

type MaskinportenReconciler struct {
	common.Reconciler
}

func NewReconciler(reconciler common.Reconciler) *MaskinportenReconciler {
	return &MaskinportenReconciler{Reconciler: reconciler}
}

// +kubebuilder:rbac:groups=nais.io,resources=MaskinportenClients,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=nais.io,resources=MaskinportenClients/status,verbs=get;update;patch;create
// +kubebuilder:rbac:groups=*,resources=events,verbs=get;list;watch;create;update

func (r *MaskinportenReconciler) Reconcile(req ctrl.Request) (ctrl.Result, error) {
	return r.Reconciler.Reconcile(req, &nais_io_v1.MaskinportenClient{})
}

func (r *MaskinportenReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&nais_io_v1.MaskinportenClient{}).
		Complete(r)
}
