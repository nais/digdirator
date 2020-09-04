package secrets

import (
	"context"
	"encoding/json"
	"fmt"
	"gopkg.in/square/go-jose.v2"

	v1 "github.com/nais/digdirator/api/v1"
	"github.com/nais/digdirator/pkg/labels"
	"github.com/nais/digdirator/pkg/pods"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
)

const (
	JwkKey = "IDPORTEN_CLIENT_JWK"
)

// +kubebuilder:rbac:groups=*,resources=secrets,verbs=get;list;watch;create;delete;update;patch

func CreateOrUpdate(ctx context.Context, instance *v1.IDPortenClient, cli client.Client, scheme *runtime.Scheme, jwk jose.JSONWebKey) (controllerutil.OperationResult, error) {
	spec, err := spec(instance, jwk)
	if err != nil {
		return controllerutil.OperationResultNone, fmt.Errorf("creating secretSpec object: %w", err)
	}

	if err := ctrl.SetControllerReference(instance, spec, scheme); err != nil {
		return controllerutil.OperationResultNone, fmt.Errorf("setting controller reference: %w", err)
	}

	err = cli.Create(ctx, spec)
	res := controllerutil.OperationResultCreated

	if errors.IsAlreadyExists(err) {
		err = cli.Update(ctx, spec)
		res = controllerutil.OperationResultUpdated
	}

	if err != nil {
		return controllerutil.OperationResultNone, fmt.Errorf("applying secretSpec: %w", err)
	}
	return res, nil
}

func GetManaged(ctx context.Context, instance *v1.IDPortenClient, reader client.Reader) (*Lists, error) {
	// fetch all application pods for this app
	podList, err := pods.GetForApplication(ctx, instance, reader)
	if err != nil {
		return nil, err
	}

	// fetch all managed secrets
	allSecrets, err := getAll(ctx, instance, reader)
	if err != nil {
		return nil, err
	}

	// find intersect between secrets in use by application pods and all managed secrets
	podSecrets := podSecretLists(allSecrets, *podList)
	return &podSecrets, nil
}

func Delete(ctx context.Context, secret corev1.Secret, cli client.Client) error {
	if err := cli.Delete(ctx, &secret); err != nil {
		return fmt.Errorf("deleting unused secret: %w", err)
	}
	return nil
}

func getAll(ctx context.Context, instance *v1.IDPortenClient, reader client.Reader) (corev1.SecretList, error) {
	var list corev1.SecretList
	mLabels := client.MatchingLabels{
		labels.AppLabelKey:  instance.GetName(),
		labels.TypeLabelKey: labels.TypeLabelValue,
	}
	if err := reader.List(ctx, &list, client.InNamespace(instance.Namespace), mLabels); err != nil {
		return list, err
	}
	return list, nil
}

func spec(instance *v1.IDPortenClient, jwk jose.JSONWebKey) (*corev1.Secret, error) {
	data, err := stringData(jwk)
	if err != nil {
		return nil, err
	}
	return &corev1.Secret{
		TypeMeta: metav1.TypeMeta{
			Kind:       "Secret",
			APIVersion: "v1",
		},
		ObjectMeta: objectMeta(instance),
		StringData: data,
		Type:       corev1.SecretTypeOpaque,
	}, nil
}

func objectMeta(instance *v1.IDPortenClient) metav1.ObjectMeta {
	return metav1.ObjectMeta{
		Name:      instance.Spec.SecretName,
		Namespace: instance.Namespace,
		Labels:    labels.DefaultLabels(instance),
	}
}

// todo
func stringData(jwk jose.JSONWebKey) (map[string]string, error) {
	jwkJson, err := json.Marshal(jwk)
	if err != nil {
		return nil, fmt.Errorf("marshalling jwk: %w", err)
	}
	return map[string]string{
		JwkKey: string(jwkJson),
	}, nil
}
