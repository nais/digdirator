package secrets

import (
	"context"
	"fmt"
	v1 "github.com/nais/digdirator/api/v1"
	"github.com/nais/digdirator/controllers"
	"github.com/nais/digdirator/pkg/config"
	"github.com/nais/digdirator/pkg/labels"
	"github.com/nais/digdirator/pkg/pods"
	"github.com/spf13/viper"
	"gopkg.in/square/go-jose.v2"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
	"strings"
)

const (
	IDPortenJwkKey           = "IDPORTEN_CLIENT_JWK"
	IDPortenClientID         = "IDPORTEN_CLIENT_ID"
	IDPortenWellKnownURL     = "IDPORTEN_WELL_KNOWN_URL"
	IDPortenRedirectURI      = "IDPORTEN_REDIRECT_URI"
	MaskinportenJwkKey       = "MASKINPORTEN_CLIENT_JWK"
	MaskinportenClientID     = "MASKINPORTEN_CLIENT_ID"
	MaskinportenWellKnownURL = "MASKINPORTEN_WELL_KNOWN_URL"
	MaskinportenScopes       = "MASKINPORTEN_SCOPES"
)

// +kubebuilder:rbac:groups=*,resources=secrets,verbs=get;list;watch;create;delete;update;patch

func CreateOrUpdateIdporten(ctx context.Context, instance *v1.IDPortenClient, cli client.Client, scheme *runtime.Scheme, jwk jose.JSONWebKey) (controllerutil.OperationResult, error) {
	spec, err := IdportenSpec(instance, jwk)
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

func CreateOrUpdateMaskinporten(ctx context.Context, instance *v1.MaskinportenClient, cli client.Client, scheme *runtime.Scheme, jwk jose.JSONWebKey) (controllerutil.OperationResult, error) {
	spec, err := MaskinportenSpec(instance, jwk)
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

func GetManaged(ctx context.Context, instance controllers.Instance, reader client.Reader) (*Lists, error) {
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

func getAll(ctx context.Context, instance controllers.Instance, reader client.Reader) (corev1.SecretList, error) {
	var list corev1.SecretList
	var mLabels client.MatchingLabels

	switch instance.(type) {
	case *v1.IDPortenClient:
		mLabels = labels.IDPortenLabels(instance)
	case *v1.MaskinportenClient:
		mLabels = labels.MaskinportenLabels(instance)
	default:
		return list, fmt.Errorf("instance does not implement 'controllers.Instance'")
	}

	if err := reader.List(ctx, &list, client.InNamespace(instance.NameSpace()), mLabels); err != nil {
		return list, err
	}
	return list, nil
}

func IdportenSpec(instance *v1.IDPortenClient, jwk jose.JSONWebKey) (*corev1.Secret, error) {
	data, err := IdportenStringData(jwk, instance)
	if err != nil {
		return nil, err
	}
	return &corev1.Secret{
		TypeMeta: metav1.TypeMeta{
			Kind:       "Secret",
			APIVersion: "v1",
		},
		ObjectMeta: ObjectMeta(controllers.Instance(instance), labels.IDPortenLabels(instance)),
		StringData: data,
		Type:       corev1.SecretTypeOpaque,
	}, nil
}

func MaskinportenSpec(instance *v1.MaskinportenClient, jwk jose.JSONWebKey) (*corev1.Secret, error) {
	data, err := MaskinportenStringData(jwk, instance)
	if err != nil {
		return nil, err
	}
	return &corev1.Secret{
		TypeMeta: metav1.TypeMeta{
			Kind:       "Secret",
			APIVersion: "v1",
		},
		ObjectMeta: ObjectMeta(controllers.Instance(instance), labels.MaskinportenLabels(instance)),
		StringData: data,
		Type:       corev1.SecretTypeOpaque,
	}, nil
}

func ObjectMeta(instance controllers.Instance, labels map[string]string) metav1.ObjectMeta {
	return metav1.ObjectMeta{
		Name:      instance.SecretName(),
		Namespace: instance.NameSpace(),
		Labels:    labels,
	}
}

func IdportenStringData(jwk jose.JSONWebKey, instance *v1.IDPortenClient) (map[string]string, error) {
	wellKnownURL := viper.GetString(config.DigDirAuthBaseURL) + "/idporten-oidc-provider/.well-known/openid-configuration"
	jwkJson, err := jwk.MarshalJSON()
	if err != nil {
		return nil, fmt.Errorf("marshalling JWK: %w", err)
	}
	return map[string]string{
		IDPortenJwkKey:       string(jwkJson),
		IDPortenWellKnownURL: wellKnownURL,
		IDPortenClientID:     instance.StatusClientID(),
		IDPortenRedirectURI:  instance.Spec.RedirectURI,
	}, nil
}

func MaskinportenStringData(jwk jose.JSONWebKey, instance *v1.MaskinportenClient) (map[string]string, error) {
	wellKnownURL := viper.GetString(config.DigDirMaskinportenBaseURL) + "/.well-known/oauth-authorization-server"
	jwkJson, err := jwk.MarshalJSON()
	if err != nil {
		return nil, fmt.Errorf("marshalling JWK: %w", err)
	}
	return map[string]string{
		MaskinportenJwkKey:       string(jwkJson),
		MaskinportenWellKnownURL: wellKnownURL,
		MaskinportenClientID:     instance.StatusClientID(),
		MaskinportenScopes:       ToScopesString(instance.Spec.Scopes),
	}, nil
}

func ToScopesString(scopes []string) string {
	return strings.Join(scopes[:], " ")
}
