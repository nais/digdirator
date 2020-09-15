package namespaces

import (
	"context"
	"fmt"
	corev1 "k8s.io/api/core/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

// +kubebuilder:rbac:groups=*,resources=namespaces,verbs=get;list;watch

func GetAll(ctx context.Context, reader client.Reader) (corev1.NamespaceList, error) {
	var namespaces corev1.NamespaceList
	if err := reader.List(ctx, &namespaces); err != nil {
		return namespaces, fmt.Errorf("listing namespaces: %w", err)
	}
	return namespaces, nil
}
