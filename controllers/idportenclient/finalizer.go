package idportenclient

import (
	"fmt"
	ctrl "sigs.k8s.io/controller-runtime"

	corev1 "k8s.io/api/core/v1"
)

const FinalizerName string = "finalizer.digdirator.nais.io"

// Finalizers allow the controller to implement an asynchronous pre-delete hook

type finalizer struct {
	*Reconciler
}

func (r *Reconciler) finalizer() finalizer {
	return finalizer{r}
}

func (f finalizer) register(tx transaction) (ctrl.Result, error) {
	if !tx.instance.HasFinalizer(FinalizerName) {
		tx.log.Info("finalizer for object not found, registering...")
		tx.instance.AddFinalizer(FinalizerName)
		if err := f.Update(tx.ctx, tx.instance); err != nil {
			return ctrl.Result{}, fmt.Errorf("error when registering finalizer: %w", err)
		}
		f.Recorder.Event(tx.instance, corev1.EventTypeNormal, "Added", "Object finalizer is added")
	}
	return ctrl.Result{}, nil
}

func (f finalizer) process(tx transaction) (ctrl.Result, error) {
	if tx.instance.HasFinalizer(FinalizerName) {
		tx.log.Info("finalizer triggered, deleting resources...")

		// todo - delete client from idporten

		tx.instance.RemoveFinalizer(FinalizerName)
		if err := f.Update(tx.ctx, tx.instance); err != nil {
			return ctrl.Result{}, fmt.Errorf("failed to remove finalizer from list: %w", err)
		}
	}
	return ctrl.Result{}, nil
}
