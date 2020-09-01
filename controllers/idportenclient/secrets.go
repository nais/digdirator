package idportenclient

import (
	"fmt"
	"github.com/nais/digdirator/pkg/secrets"
	corev1 "k8s.io/api/core/v1"
)

func (r *Reconciler) createOrUpdateSecrets(tx transaction) error {
	tx.log.Infof("processing secret with name '%s'...", tx.instance.Spec.SecretName)
	res, err := secrets.CreateOrUpdate(tx.ctx, tx.instance, r.Client, r.Scheme)
	if err != nil {
		return fmt.Errorf("failed to create or update secret: %w", err)
	}
	tx.log.Infof("secret '%s' %s", tx.instance.Spec.SecretName, res)
	return nil
}

func (r *Reconciler) deleteUnusedSecrets(tx transaction, unused corev1.SecretList) error {
	for _, oldSecret := range unused.Items {
		if oldSecret.Name == tx.instance.Spec.SecretName {
			continue
		}
		tx.log.Infof("deleting unused secret '%s'...", oldSecret.Name)
		if err := secrets.Delete(tx.ctx, oldSecret, r.Client); err != nil {
			return err
		}
	}
	return nil
}
