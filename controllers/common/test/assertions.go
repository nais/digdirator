package test

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"

	"github.com/nais/digdirator/pkg/clients"
)

func ResourceExists(cli client.Client, key client.ObjectKey, instance client.Object) func() bool {
	return func() bool {
		err := cli.Get(context.Background(), key, instance)
		return !errors.IsNotFound(err)
	}
}

func ResourceDoesNotExist(cli client.Client, key client.ObjectKey, instance client.Object) func() bool {
	return func() bool {
		err := cli.Get(context.Background(), key, instance)
		return errors.IsNotFound(err)
	}
}

func ContainsOwnerRef(refs []metav1.OwnerReference, owner clients.Instance) bool {
	expected := metav1.OwnerReference{
		APIVersion: owner.GroupVersionKind().GroupVersion().String(),
		Kind:       owner.GroupVersionKind().Kind,
		Name:       owner.GetName(),
		UID:        owner.GetUID(),
	}
	for _, ref := range refs {
		sameApiVersion := ref.APIVersion == expected.APIVersion
		sameKind := ref.Kind == expected.Kind
		sameName := ref.Name == expected.Name
		sameUID := ref.UID == expected.UID
		if sameApiVersion && sameKind && sameName && sameUID {
			return true
		}
	}
	return false
}

func AssertSecretExists(t *testing.T, cli client.Client, name string, namespace string, instance clients.Instance, assertions func(*corev1.Secret, clients.Instance)) {
	key := client.ObjectKey{
		Namespace: namespace,
		Name:      name,
	}
	a := &corev1.Secret{}
	err := cli.Get(context.Background(), key, a)
	assert.NoError(t, err)

	assert.True(t, ContainsOwnerRef(a.GetOwnerReferences(), instance), "Secret should contain ownerReference")

	assertions(a, instance)
}
