package common

import (
	"fmt"

	nais_io_v1 "github.com/nais/liberator/pkg/apis/nais.io/v1"
	"github.com/nais/liberator/pkg/kubernetes"
	"gopkg.in/square/go-jose.v2"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
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
	Reconciler
	secretName string
}

func (r Reconciler) secrets(transaction *Transaction) secretsClient {
	return secretsClient{
		Transaction: transaction,
		Reconciler:  r,
		secretName:  clients.GetSecretName(transaction.Instance),
	}
}

func (s secretsClient) CreateOrUpdate(jwk jose.JSONWebKey) error {
	s.Logger.Infof("processing secret with name '%s'...", s.secretName)

	labels := clients.MakeLabels(s.Instance)

	objectMeta := kubernetes.ObjectMeta(s.secretName, s.Instance.GetNamespace(), labels)
	objectMeta.SetAnnotations(map[string]string{
		StakaterReloaderKeyAnnotation: "true",
	})

	stringData, err := secretData(s.Instance, jwk, s.Config)
	if err != nil {
		return fmt.Errorf("while creating secret data: %w", err)
	}

	spec := kubernetes.OpaqueSecret(objectMeta, stringData)

	if err := ctrl.SetControllerReference(s.Instance, &spec, s.Scheme); err != nil {
		return fmt.Errorf("setting controller reference: %w", err)
	}

	err = s.Client.Create(s.Ctx, &spec)
	res := controllerutil.OperationResultCreated

	if errors.IsAlreadyExists(err) {
		err = s.Client.Update(s.Ctx, &spec)
		res = controllerutil.OperationResultUpdated
	}
	if err != nil {
		return fmt.Errorf("applying secretSpec: %w", err)
	}
	s.Logger.Infof("secret '%s' %s", s.secretName, res)
	return nil
}

func (s secretsClient) GetManaged() (*kubernetes.SecretLists, error) {
	// fetch all application pods for this app
	podList, err := kubernetes.ListPodsForApplication(s.Ctx, s.Reader, s.Instance.GetName(), s.Instance.GetNamespace())
	if err != nil {
		return nil, err
	}

	// fetch all managed secrets
	var allSecrets corev1.SecretList
	opts := []client.ListOption{
		client.InNamespace(s.Instance.GetNamespace()),
		client.MatchingLabels(clients.MakeLabels(s.Instance)),
	}
	if err := s.Reader.List(s.Ctx, &allSecrets, opts...); err != nil {
		return nil, err
	}

	// find intersect between secrets in use by application pods and all managed secrets
	podSecrets := kubernetes.ListUsedAndUnusedSecretsForPods(allSecrets, podList)
	return &podSecrets, nil
}

func (s secretsClient) DeleteUnused(unused corev1.SecretList) error {
	for i, oldSecret := range unused.Items {
		if oldSecret.Name == s.secretName {
			continue
		}
		s.Logger.Infof("deleting unused secret '%s'...", oldSecret.Name)
		if err := s.Client.Delete(s.Ctx, &unused.Items[i]); err != nil {
			return fmt.Errorf("deleting unused secret: %w", err)
		}
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
