package pods

import (
	"context"
	"github.com/nais/digdirator/controllers"
	"github.com/nais/digdirator/pkg/labels"
	corev1 "k8s.io/api/core/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

// +kubebuilder:rbac:groups=*,resources=pods,verbs=get;list;watch

func GetForApplication(ctx context.Context, instance controllers.Instance, reader client.Reader) (*corev1.PodList, error) {
	selector := client.MatchingLabels{
		labels.AppLabelKey: instance.ClientName(),
	}
	namespace := client.InNamespace(instance.NameSpace())
	podList := &corev1.PodList{}
	err := reader.List(ctx, podList, selector, namespace)
	if err != nil {
		return nil, err
	}
	return podList, nil
}
