package common

import (
	"fmt"

	"github.com/go-jose/go-jose/v4"
	"github.com/go-logr/logr"
	nais_io_v1 "github.com/nais/liberator/pkg/apis/nais.io/v1"
	"github.com/nais/liberator/pkg/kubernetes"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"

	"github.com/nais/digdirator/pkg/clients"
	"github.com/nais/digdirator/pkg/config"
	"github.com/nais/digdirator/pkg/secrets"
)

// +kubebuilder:rbac:groups=*,resources=secrets,verbs=get;list;watch;create;delete;update;patch

const (
	StakaterReloaderKeyAnnotation = "reloader.stakater.com/match"
)

type secretsClient struct {
	*Transaction
	*Reconciler
	secretName string
	log        logr.Logger
}

func (r *Reconciler) secrets(transaction *Transaction) secretsClient {
	return secretsClient{
		Transaction: transaction,
		Reconciler:  r,
		secretName:  clients.GetSecretName(transaction.Instance),
		log:         ctrl.LoggerFrom(transaction.Ctx).WithValues("subsystem", "secrets"),
	}
}

func (s secretsClient) CreateOrUpdate(jwk jose.JSONWebKey) error {
	name := s.secretName
	namespace := s.Instance.GetNamespace()
	s.log.V(4).Info(fmt.Sprintf("processing secret %q...", name))

	stringData, err := secretData(s.Instance, jwk, s.Reconciler.Config)
	if err != nil {
		return fmt.Errorf("creating secret data: %w", err)
	}

	data := make(map[string][]byte)
	for key, value := range stringData {
		data[key] = []byte(value)
	}

	target := &corev1.Secret{ObjectMeta: metav1.ObjectMeta{
		Name:      name,
		Namespace: namespace,
	}}

	res, err := controllerutil.CreateOrUpdate(s.Ctx, s.Client, target, func() error {
		target.SetAnnotations(map[string]string{
			StakaterReloaderKeyAnnotation: "true",
		})
		target.SetLabels(clients.MakeLabels(s.Instance))
		target.Data = data

		return ctrl.SetControllerReference(s.Instance, target, s.Scheme)
	})
	if err != nil {
		return fmt.Errorf("creating or updating secret %q: %w", name, err)
	}

	s.log.Info(fmt.Sprintf("secret %q %s", name, res))
	return nil
}

func (s secretsClient) GetManaged() (kubernetes.SecretLists, error) {
	objectKey := client.ObjectKey{
		Name:      s.Instance.GetName(),
		Namespace: s.Instance.GetNamespace(),
	}
	secretLabels := clients.MakeLabels(s.Instance)
	return kubernetes.ListSecretsForApplication(s.Ctx, s.Reader, objectKey, secretLabels)
}

func (s secretsClient) DeleteUnused(unused corev1.SecretList) error {
	for i, oldSecret := range unused.Items {
		if oldSecret.Name == s.secretName {
			continue
		}

		s.log.V(4).Info(fmt.Sprintf("deleting unused secret %q...", oldSecret.Name))
		if err := s.Client.Delete(s.Ctx, &unused.Items[i]); err != nil {
			return fmt.Errorf("deleting unused secret: %w", err)
		}
		s.log.Info(fmt.Sprintf("deleted unused secret %q", oldSecret.Name))
	}
	return nil
}

func secretData(instance clients.Instance, jwk jose.JSONWebKey, config *config.Config) (map[string]string, error) {
	var stringData map[string]string
	var err error

	switch v := instance.(type) {
	case *nais_io_v1.IDPortenClient:
		stringData, err = secrets.IDPortenClientSecretData(v, jwk, config)
	case *nais_io_v1.MaskinportenClient:
		stringData, err = secrets.MaskinportenClientSecretData(v, jwk, config)
	}

	if err != nil {
		return nil, err
	}

	return stringData, nil
}
