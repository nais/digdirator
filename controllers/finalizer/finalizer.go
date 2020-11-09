package finalizer

import (
	"fmt"
	"github.com/nais/digdirator/controllers/common"
	"github.com/nais/digdirator/controllers/common/reconciler"
	"github.com/nais/digdirator/controllers/common/transaction"
	corev1 "k8s.io/api/core/v1"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
)

const FinalizerName string = "finalizer.digdirator.nais.io"

// Finalizers allow the controller to implement an asynchronous pre-delete hook

type Finalizer struct {
	reconciler.Reconciler
	transaction.Transaction
}

func Client(reconciler reconciler.Reconciler, transaction transaction.Transaction) Finalizer {
	return Finalizer{Reconciler: reconciler, Transaction: transaction}
}

func (f Finalizer) Register() (ctrl.Result, error) {
	if !common.HasFinalizer(f.Instance, FinalizerName) {
		f.Logger.Info("finalizer for object not found, registering...")
		controllerutil.AddFinalizer(f.Instance, FinalizerName)
		if err := f.Client.Update(f.Ctx, f.Instance); err != nil {
			return ctrl.Result{}, fmt.Errorf("registering finalizer: %w", err)
		}
		f.Recorder.Event(f.Instance, corev1.EventTypeNormal, "Added", "Object finalizer is added")
	}
	return ctrl.Result{}, nil
}

func (f Finalizer) Process() (ctrl.Result, error) {
	if !common.HasFinalizer(f.Instance, FinalizerName) {
		return ctrl.Result{}, nil
	}

	f.Logger.Info("finalizer triggered, deleting resources...")

	if len(f.Instance.StatusClientID()) == 0 {
		return ctrl.Result{}, nil
	}
	if err := f.Client.Delete(f.Ctx, f.Instance); err != nil {
		return ctrl.Result{}, fmt.Errorf("deleting client from ID-porten: %w", err)
	}

	controllerutil.RemoveFinalizer(f.Instance, FinalizerName)
	if err := f.Client.Update(f.Ctx, f.Instance); err != nil {
		return ctrl.Result{}, fmt.Errorf("removing finalizer from list: %w", err)
	}

	f.Logger.Info("finalizer processing completed")
	return ctrl.Result{}, nil
}
