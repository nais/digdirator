package idportenclient

import (
	"fmt"
	"github.com/nais/digdirator/pkg/metrics"
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

func (f finalizer) register(tx *transaction) (ctrl.Result, error) {
	if !tx.instance.HasFinalizer(FinalizerName) {
		tx.log.Info("finalizer for object not found, registering...")
		tx.instance.AddFinalizer(FinalizerName)
		if err := f.Update(tx.ctx, tx.instance); err != nil {
			return ctrl.Result{}, fmt.Errorf("registering finalizer: %w", err)
		}
		f.Recorder.Event(tx.instance, corev1.EventTypeNormal, "Added", "Object finalizer is added")
	}
	return ctrl.Result{}, nil
}

func (f finalizer) process(tx *transaction) (ctrl.Result, error) {
	if tx.instance.HasFinalizer(FinalizerName) {
		tx.log.Info("finalizer triggered, deleting resources...")

		if len(tx.instance.Status.ClientID) > 0 {
			if err := tx.digdirClient.Delete(tx.ctx, tx.instance.Status.ClientID); err != nil {
				return ctrl.Result{}, fmt.Errorf("deleting client from ID-porten: %w", err)
			}
		}

		tx.instance.RemoveFinalizer(FinalizerName)
		if err := f.Update(tx.ctx, tx.instance); err != nil {
			return ctrl.Result{}, fmt.Errorf("removing finalizer from list: %w", err)
		}

		tx.log.Info("resources deleted")
		metrics.IncWithNamespaceLabel(metrics.IDPortenClientsDeletedCount, tx.instance.Namespace)
	}
	return ctrl.Result{}, nil
}
