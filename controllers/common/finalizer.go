package common

import (
	"fmt"
	"github.com/nais/digdirator/pkg/metrics"
	finalizer2 "github.com/nais/liberator/pkg/finalizer"
	corev1 "k8s.io/api/core/v1"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
)

const FinalizerName string = "finalizer.digdirator.nais.io"

// Finalizers allow the controller to implement an asynchronous pre-delete hook
type finalizer struct {
	Reconciler
	*Transaction
}

func (r Reconciler) finalizer(transaction *Transaction) finalizer {
	return finalizer{Reconciler: r, Transaction: transaction}
}

func (f finalizer) Register() (ctrl.Result, error) {
	if !finalizer2.HasFinalizer(f.Instance, FinalizerName) {
		f.Logger.Info("finalizer for object not found, registering...")
		controllerutil.AddFinalizer(f.Instance, FinalizerName)
		if err := f.Client.Update(f.Ctx, f.Instance); err != nil {
			return ctrl.Result{}, fmt.Errorf("registering finalizer: %w", err)
		}
		f.reportEvent(f.Transaction, corev1.EventTypeNormal, EventAddedFinalizer, "Object finalizer is added")
	}
	return ctrl.Result{}, nil
}

func (f finalizer) Process() (ctrl.Result, error) {
	if !finalizer2.HasFinalizer(f.Instance, FinalizerName) {
		return ctrl.Result{}, nil
	}

	f.Logger.Info("finalizer triggered, deleting resources...")

	clientRegistration, err := f.DigdirClient.ClientExists(f.Instance, f.Ctx)
	if err != nil {
		return ctrl.Result{}, err
	}

	if clientRegistration != nil {
		f.Logger.Info("client does not exist, skipping deletion...")

		if err := f.DigdirClient.Delete(f.Ctx, f.Instance.GetStatus().GetClientID()); err != nil {
			return ctrl.Result{}, fmt.Errorf("deleting client from ID-porten: %w", err)
		}
		f.reportEvent(f.Transaction, corev1.EventTypeNormal, EventDeletedInDigDir, "Client deleted in Digdir")
	}

	controllerutil.RemoveFinalizer(f.Instance, FinalizerName)
	if err := f.Client.Update(f.Ctx, f.Instance); err != nil {
		return ctrl.Result{}, fmt.Errorf("removing finalizer from list: %w", err)
	}
	f.reportEvent(f.Transaction, corev1.EventTypeNormal, EventDeletedFinalizer, "Object finalizer is removed")
	metrics.IncClientsDeleted(f.Instance)

	f.Logger.Info("finalizer processing completed")
	return ctrl.Result{}, nil
}
