package common

import (
	"fmt"

	naisiov1 "github.com/nais/liberator/pkg/apis/nais.io/v1"
	corev1 "k8s.io/api/core/v1"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"

	"github.com/nais/digdirator/pkg/clients"
	"github.com/nais/digdirator/pkg/metrics"
)

const (
	FinalizerName string = "finalizer.digdirator.nais.io"
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
		f.Logger.Info("finalizer for object not found, registering...")

		err := f.updateInstance(f.Ctx, f.Instance, func(existing clients.Instance) error {
			controllerutil.AddFinalizer(existing, FinalizerName)
			return f.Client.Update(f.Ctx, existing)
		})
		if err != nil {
			return ctrl.Result{}, fmt.Errorf("registering finalizer: %w", err)
		}
		f.reportEvent(f.Transaction, corev1.EventTypeNormal, EventAddedFinalizer, "Object finalizer is added")
	}
	return ctrl.Result{}, nil
}

func (f finalizer) Process() (ctrl.Result, error) {
	if !controllerutil.ContainsFinalizer(f.Instance, FinalizerName) {
		return ctrl.Result{}, nil
	}

	f.Logger.Info("finalizer triggered...")

	clientRegistration, err := f.DigdirClient.ClientExists(f.Instance, f.Ctx, f.Config.ClusterName)
	if err != nil {
		return ctrl.Result{}, err
	}

	clientExists := clientRegistration != nil

	if clientExists {
		f.Logger.Info("delete annotation set, deleting client from DigDir...")
		if err := f.DigdirClient.Delete(f.Ctx, f.Instance.GetStatus().GetClientID()); err != nil {
			return ctrl.Result{}, fmt.Errorf("deleting client from ID-porten: %w", err)
		}
		f.reportEvent(f.Transaction, corev1.EventTypeNormal, EventDeletedInDigDir, "Client deleted in Digdir")
	}

	switch instance := f.Instance.(type) {
	case *naisiov1.MaskinportenClient:
		exposedScopes := instance.GetExposedScopes()
		scopes := f.Reconciler.scopes(f.Transaction)

		if exposedScopes != nil {
			if err := scopes.Finalize(exposedScopes); err != nil {
				return ctrl.Result{}, fmt.Errorf("deleting scope from Maskinporten: %w", err)
			}
		}
	}

	err = f.updateInstance(f.Ctx, f.Instance, func(existing clients.Instance) error {
		controllerutil.RemoveFinalizer(existing, FinalizerName)
		return f.Client.Update(f.Ctx, existing)
	})
	if err != nil {
		return ctrl.Result{}, fmt.Errorf("removing finalizer from list: %w", err)
	}

	f.reportEvent(f.Transaction, corev1.EventTypeNormal, EventDeletedFinalizer, "Object finalizer is removed")
	metrics.IncClientsDeleted(f.Instance)

	f.Logger.Info("finalizer processing completed")
	return ctrl.Result{}, nil
}
