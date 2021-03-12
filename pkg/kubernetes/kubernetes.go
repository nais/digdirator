package kubernetes

import (
	"context"
	"fmt"

	corev1 "k8s.io/api/core/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

func ListNamespaces(ctx context.Context, reader client.Reader, opts ...client.ListOption) (corev1.NamespaceList, error) {
	var namespaces corev1.NamespaceList
	if err := reader.List(ctx, &namespaces, opts...); err != nil {
		return namespaces, fmt.Errorf("listing namespaces: %w", err)
	}
	return namespaces, nil
}

func ListSharedNamespaces(ctx context.Context, reader client.Reader) (corev1.NamespaceList, error) {
	opts := client.MatchingLabels{
		"shared": "true",
	}
	return ListNamespaces(ctx, reader, opts)
}
