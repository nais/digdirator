package pods

import (
	"context"
	"github.com/nais/digdirator/api/v1"
	"github.com/nais/digdirator/pkg/labels"
	corev1 "k8s.io/api/core/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

// +kubebuilder:rbac:groups=*,resources=pods,verbs=get;list;watch

func GetForApplication(ctx context.Context, instance v1.Instance, reader client.Reader) (*corev1.PodList, error) {
	selector := client.MatchingLabels{
		labels.AppLabelKey: instance.GetName(),
	}
	namespace := client.InNamespace(instance.GetNamespace())
	podList := &corev1.PodList{}
	err := reader.List(ctx, podList, selector, namespace)
	if err != nil {
		return nil, err
	}
	return podList, nil
}
