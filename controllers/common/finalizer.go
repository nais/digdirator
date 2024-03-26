package common

import (
	"fmt"

	naisiov1 "github.com/nais/liberator/pkg/apis/nais.io/v1"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"

	"github.com/nais/digdirator/pkg/clients"
	"github.com/nais/digdirator/pkg/metrics"
)

const (
	FinalizerName      string = "finalizer.digdirator.nais.io"
	PreserveAnnotation string = "digdir.nais.io/preserve"
)

// Finalizers allow the controller to implement an asynchronous pre-delete hook
type finalizer struct {
	Reconciler
	*Transaction
}

func (r Reconciler) finalizer(transaction *Transaction) finalizer {
	return finalizer{Reconciler: r, Transaction: transaction}
}

func (f finalizer) Register() (ctrl.Result, error) {
	if !controllerutil.ContainsFinalizer(f.Instance, FinalizerName) {
		f.Logger.Debug("finalizer for object not found, registering...")

		err := f.updateInstance(f.Ctx, f.Instance, func(existing clients.Instance) error {
			controllerutil.AddFinalizer(existing, FinalizerName)
			return f.Client.Update(f.Ctx, existing)
		})
		if err != nil {
			return ctrl.Result{}, fmt.Errorf("registering finalizer: %w", err)
		}
	}
	return ctrl.Result{}, nil
}

func (f finalizer) Process() (ctrl.Result, error) {
	if !controllerutil.ContainsFinalizer(f.Instance, FinalizerName) {
		return ctrl.Result{}, nil
	}

	f.Logger.Debug("finalizer triggered...")

	exists, err := f.DigdirClient.Exists(f.Ctx, f.Instance, f.Reconciler.Config.ClusterName)
	if err != nil {
		return ctrl.Result{}, fmt.Errorf("checking client existence: %w", err)
	}

	if exists {
		if hasPreserveAnnotation(f.Instance.GetAnnotations()) {
			f.Logger.Info("preserve annotation set, skipping external deletion...")
		} else {
			f.Logger.Info("deleting client from DigDir...")
			if err := f.DigdirClient.Delete(f.Ctx, f.Instance.GetStatus().GetClientID()); err != nil {
				return ctrl.Result{}, fmt.Errorf("deleting client: %w", err)
			}
		}
	}

	switch instance := f.Instance.(type) {
	case *naisiov1.MaskinportenClient:
		exposedScopes := instance.GetExposedScopes()
		scopes := f.Reconciler.scopes(f.Transaction)

		if exposedScopes != nil {
			if err := scopes.Finalize(exposedScopes); err != nil {
				return ctrl.Result{}, fmt.Errorf("deleting Maskinporten scope: %w", err)
			}
		}
	}

	err = f.updateInstance(f.Ctx, f.Instance, func(existing clients.Instance) error {
		controllerutil.RemoveFinalizer(existing, FinalizerName)
		return f.Client.Update(f.Ctx, existing)
	})
	if err != nil {
		return ctrl.Result{}, fmt.Errorf("removing finalizer: %w", err)
	}

	metrics.IncClientsDeleted(f.Instance)

	f.Logger.Debug("finalizer processed")
	return ctrl.Result{}, nil
}

func hasPreserveAnnotation(annotations map[string]string) bool {
	value, found := annotations[PreserveAnnotation]
	return found && value == "true"
}
