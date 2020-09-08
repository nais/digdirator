package secrets

import (
	"context"
	"fmt"
	"github.com/nais/digdirator/pkg/config"
	"github.com/spf13/viper"
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
	JwkKey       = "IDPORTEN_CLIENT_JWK"
	ClientID     = "IDPORTEN_CLIENT_ID"
	WellKnownURL = "IDPORTEN_WELL_KNOWN_URL"
)

// +kubebuilder:rbac:groups=*,resources=secrets,verbs=get;list;watch;create;delete;update;patch

func CreateOrUpdate(ctx context.Context, instance *v1.IDPortenClient, cli client.Client, scheme *runtime.Scheme, jwk jose.JSONWebKey) (controllerutil.OperationResult, error) {
	spec, err := Spec(instance, jwk)
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

func Spec(instance *v1.IDPortenClient, jwk jose.JSONWebKey) (*corev1.Secret, error) {
	clientID := instance.Status.ClientID
	data, err := StringData(jwk, clientID)
	if err != nil {
		return nil, err
	}
	return &corev1.Secret{
		TypeMeta: metav1.TypeMeta{
			Kind:       "Secret",
			APIVersion: "v1",
		},
		ObjectMeta: ObjectMeta(instance),
		StringData: data,
		Type:       corev1.SecretTypeOpaque,
	}, nil
}

func ObjectMeta(instance *v1.IDPortenClient) metav1.ObjectMeta {
	return metav1.ObjectMeta{
		Name:      instance.Spec.SecretName,
		Namespace: instance.Namespace,
		Labels:    labels.DefaultLabels(instance),
	}
}

func StringData(jwk jose.JSONWebKey, clientID string) (map[string]string, error) {
	wellKnownURL := viper.GetString(config.DigDirAuthBaseURL) + "/idporten-oidc-provider/.well-known/openid-configuration"
	jwkJson, err := jwk.MarshalJSON()
	if err != nil {
		return nil, fmt.Errorf("marshalling JWK: %w", err)
	}
	return map[string]string{
		JwkKey:       string(jwkJson),
		WellKnownURL: wellKnownURL,
		ClientID:     clientID,
	}, nil
}
