package secrets

import (
	"context"
	"fmt"
	v1 "github.com/nais/digdirator/api/v1"
	"github.com/nais/digdirator/controllers"
	"github.com/nais/digdirator/pkg/labels"
	"github.com/nais/digdirator/pkg/pods"
	"gopkg.in/square/go-jose.v2"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
)

// +kubebuilder:rbac:groups=*,resources=secrets,verbs=get;list;watch;create;delete;update;patch

func CreateOrUpdate(
	ctx context.Context,
	instance controllers.Instance,
	cli client.Client,
	scheme *runtime.Scheme,
	jwk jose.JSONWebKey,
) (controllerutil.OperationResult, error) {

	var data map[string]string
	var err error
	var objectMeta metav1.ObjectMeta
	var metav1Object metav1.Object

	switch instance.(type) {
	case *v1.IDPortenClient:
		metav1Object = instance.(*v1.IDPortenClient)
		objectMeta = ObjectMeta(instance, labels.IDPortenLabels(instance))
		data, err = IDPortenStringData(jwk, instance.(*v1.IDPortenClient))
	case *v1.MaskinportenClient:
		metav1Object = instance.(*v1.MaskinportenClient)
		objectMeta = ObjectMeta(instance, labels.MaskinportenLabels(instance))
		data, err = MaskinportenStringData(jwk, instance.(*v1.MaskinportenClient))
	default:
		return controllerutil.OperationResultNone, fmt.Errorf("instance does not implement 'controllers.Instance'")
	}

	if err != nil {
		return controllerutil.OperationResultNone, err
	}

	spec := Spec(data, objectMeta)

	if err := ctrl.SetControllerReference(metav1Object, spec, scheme); err != nil {
		return controllerutil.OperationResultNone, fmt.Errorf("setting controller reference: %w", err)
	}

	return createOrUpdate(ctx, cli, spec)
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

func createOrUpdate(ctx context.Context, cli client.Client, spec *corev1.Secret) (controllerutil.OperationResult, error) {
	err := cli.Create(ctx, spec)
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
