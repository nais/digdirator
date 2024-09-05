package common

import (
	"fmt"

	"github.com/nais/digdirator/pkg/metrics"
	naisiov1 "github.com/nais/liberator/pkg/apis/nais.io/v1"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
)

const (
	FinalizerName      string = "finalizer.digdirator.nais.io"
	PreserveAnnotation string = "digdir.nais.io/preserve"
)

func (r *Reconciler) finalize(tx *Transaction) (ctrl.Result, error) {
	if !controllerutil.ContainsFinalizer(tx.Instance, FinalizerName) {
		return ctrl.Result{}, nil
	}

	exists, err := tx.DigdirClient.Exists(tx.Ctx, tx.Instance, r.Config.ClusterName)
	if err != nil {
		return ctrl.Result{}, fmt.Errorf("finalizer: checking client existence: %w", err)
	}

	// delete client registration
	switch {
	case !exists:
		tx.Logger.Info("finalizer: client does not exist in DigDir, skipping external deletion...")
	case shouldPreserve(tx.Instance):
		tx.Logger.Info("finalizer: preserve annotation set, skipping external deletion...")
	default:
		if err := tx.DigdirClient.Delete(tx.Ctx, tx.Instance.GetStatus().ClientID); err != nil {
			return ctrl.Result{}, fmt.Errorf("deleting client: %w", err)
		}
		tx.Logger.Info("finalizer: deleted client from DigDir")
	}

	if obj, ok := tx.Instance.(*naisiov1.MaskinportenClient); ok {
		if err := r.scopes(tx).Finalize(obj.GetExposedScopes()); err != nil {
			return ctrl.Result{}, fmt.Errorf("finalizer: deleting Maskinporten scope: %w", err)
		}
	}

	controllerutil.RemoveFinalizer(tx.Instance, FinalizerName)
	err = r.Client.Update(tx.Ctx, tx.Instance)
	if err != nil {
		return ctrl.Result{}, fmt.Errorf("removing finalizer: %w", err)
	}

	metrics.IncClientsDeleted(tx.Instance)
	return ctrl.Result{}, nil
}

func markedForDeletion(o client.Object) bool {
	return !o.GetDeletionTimestamp().IsZero()
}

func shouldPreserve(o client.Object) bool {
	value, found := o.GetAnnotations()[PreserveAnnotation]
	return found && value == "true"
}
