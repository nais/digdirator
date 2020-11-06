package maskinportenclient

import (
	"fmt"
	"github.com/nais/digdirator/pkg/secrets"
	"gopkg.in/square/go-jose.v2"
	corev1 "k8s.io/api/core/v1"
)

type secretsReconciler struct {
	*Reconciler
}

func (r *Reconciler) secrets() secretsReconciler {
	return secretsReconciler{r}
}

func (s secretsReconciler) createOrUpdate(tx *transaction, jwk jose.JSONWebKey) error {
	tx.log.Infof("processing secret with name '%s'...", tx.instance.Spec.SecretName)
	res, err := secrets.CreateOrUpdate(tx.ctx, tx.instance, s.Client, s.Scheme, jwk)
	if err != nil {
		return fmt.Errorf("creating or updating secret: %w", err)
	}
	tx.log.Infof("secret '%s' %s", tx.instance.Spec.SecretName, res)
	return nil
}

func (s secretsReconciler) deleteUnused(tx *transaction, unused corev1.SecretList) error {
	for _, oldSecret := range unused.Items {
		if oldSecret.Name == tx.instance.Spec.SecretName {
			continue
		}
		tx.log.Infof("deleting unused secret '%s'...", oldSecret.Name)
		if err := secrets.Delete(tx.ctx, oldSecret, s.Client); err != nil {
			return err
		}
	}
	return nil
}
