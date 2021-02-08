package common

import (
	"fmt"
	"github.com/nais/digdirator/pkg/pods"
	"github.com/nais/digdirator/pkg/secrets"
	"gopkg.in/square/go-jose.v2"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
)

// +kubebuilder:rbac:groups=*,resources=secrets,verbs=get;list;watch;create;delete;update;patch

type secretsClient struct {
	*Transaction
	Reconciler
}

func (r Reconciler) secrets(transaction *Transaction) secretsClient {
	return secretsClient{Transaction: transaction, Reconciler: r}
}

func (s secretsClient) CreateOrUpdate(jwk jose.JSONWebKey) error {
	s.Logger.Infof("processing secret with name '%s'...", s.Instance.GetSecretName())
	spec, err := secrets.OpaqueSecret(s.Instance, jwk)
	if err != nil {
		return fmt.Errorf("creating secret spec: %w", err)
	}
	if err := ctrl.SetControllerReference(s.Instance, spec, s.Scheme); err != nil {
		return fmt.Errorf("setting controller reference: %w", err)
	}

	err = s.Client.Create(s.Ctx, spec)
	res := controllerutil.OperationResultCreated

	if errors.IsAlreadyExists(err) {
		err = s.Client.Update(s.Ctx, spec)
		res = controllerutil.OperationResultUpdated
	}
	if err != nil {
		return fmt.Errorf("applying secretSpec: %w", err)
	}
	s.Logger.Infof("secret '%s' %s", s.Instance.GetSecretName(), res)
	return nil
}

func (s secretsClient) GetManaged() (*secrets.Lists, error) {
	// fetch all application pods for this app
	podList, err := pods.GetForApplication(s.Ctx, s.Instance, s.Reader)
	if err != nil {
		return nil, err
	}

	// fetch all managed secrets
	var allSecrets corev1.SecretList
	opts := []client.ListOption{
		client.InNamespace(s.Instance.GetNamespace()),
		client.MatchingLabels(s.Instance.MakeLabels()),
	}
	if err := s.Reader.List(s.Ctx, &allSecrets, opts...); err != nil {
		return nil, err
	}

	// find intersect between secrets in use by application pods and all managed secrets
	podSecrets := secrets.PodSecretLists(allSecrets, *podList)
	return &podSecrets, nil
}

func (s secretsClient) DeleteUnused(unused corev1.SecretList) error {
	for i, oldSecret := range unused.Items {
		if oldSecret.Name == s.Instance.GetSecretName() {
			continue
		}
		s.Logger.Infof("deleting unused secret '%s'...", oldSecret.Name)
		if err := s.Client.Delete(s.Ctx, &unused.Items[i]); err != nil {
			return fmt.Errorf("deleting unused secret: %w", err)
		}
	}
	return nil
}
