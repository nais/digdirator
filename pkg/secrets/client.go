package secrets

import (
	"context"
	"fmt"
	"github.com/nais/digdirator/controllers/common"
	"github.com/nais/digdirator/pkg/pods"
	"gopkg.in/square/go-jose.v2"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/runtime"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
)

// +kubebuilder:rbac:groups=*,resources=secrets,verbs=get;list;watch;create;delete;update;patch

func CreateOrUpdate(
	ctx context.Context,
	instance common.Instance,
	cli client.Client,
	scheme *runtime.Scheme,
	jwk jose.JSONWebKey,
) (controllerutil.OperationResult, error) {
	spec, err := OpaqueSecret(instance, jwk)
	if err != nil {
		return controllerutil.OperationResultNone, err
	}
	if err := ctrl.SetControllerReference(instance, spec, scheme); err != nil {
		return controllerutil.OperationResultNone, fmt.Errorf("setting controller reference: %w", err)
	}
	return createOrUpdate(ctx, cli, spec)
}

func GetManaged(ctx context.Context, instance common.Instance, reader client.Reader) (*Lists, error) {
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

func getAll(ctx context.Context, instance common.Instance, reader client.Reader) (corev1.SecretList, error) {
	var list corev1.SecretList

	if err := reader.List(ctx, &list, client.InNamespace(instance.GetNamespace()), client.MatchingLabels(instance.Labels())); err != nil {
		return list, err
	}
	return list, nil
}
