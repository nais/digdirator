package secrets

import (
	"context"
	"fmt"

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

// +kubebuilder:rbac:groups=*,resources=secrets,verbs=get;list;watch;create;delete;update;patch

func CreateOrUpdate(ctx context.Context, instance *v1.IDPortenClient, cli client.Client, scheme *runtime.Scheme) (controllerutil.OperationResult, error) {
	spec, err := spec(instance)
	if err != nil {
		return controllerutil.OperationResultNone, fmt.Errorf("unable to create secretSpec object: %w", err)
	}

	if err := ctrl.SetControllerReference(instance, spec, scheme); err != nil {
		return controllerutil.OperationResultNone, fmt.Errorf("failed to set controller reference: %w", err)
	}

	err = cli.Create(ctx, spec)
	res := controllerutil.OperationResultCreated

	if errors.IsAlreadyExists(err) {
		err = cli.Update(ctx, spec)
		res = controllerutil.OperationResultUpdated
	}

	if err != nil {
		return controllerutil.OperationResultNone, fmt.Errorf("unable to apply secretSpec: %w", err)
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
		return fmt.Errorf("failed to delete unused secret: %w", err)
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

func spec(instance *v1.IDPortenClient) (*corev1.Secret, error) {
	data, err := stringData()
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
		Labels:    labels.Labels(instance),
	}
}

// todo
func stringData() (map[string]string, error) {
	return map[string]string{}, nil
}
