package test

import (
	"context"
	"github.com/nais/digdirator/controllers/common"
	"github.com/nais/digdirator/pkg/annotations"
	"github.com/nais/liberator/pkg/finalizer"
	"strconv"
	"testing"

	"github.com/stretchr/testify/assert"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"

	"github.com/nais/digdirator/pkg/clients"
)

func ResourceExists(cli client.Client, key client.ObjectKey, instance runtime.Object) func() bool {
	return func() bool {
		err := cli.Get(context.Background(), key, instance)
		return !errors.IsNotFound(err)
	}
}

func ResourceDoesNotExist(cli client.Client, key client.ObjectKey, instance runtime.Object) func() bool {
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

func AssertApplicationShouldNotProcess(t *testing.T, cli client.Client, key client.ObjectKey, instance clients.Instance) clients.Instance {
	assert.Eventually(t, ResourceExists(cli, key, instance), Timeout, Interval, "Client should exist")
	assert.Eventually(t, func() bool {
		_ = cli.Get(context.Background(), key, instance)

		hasCorrelationID := len(instance.GetStatus().CorrelationID) > 0
		hasFinalizer := finalizer.HasFinalizer(instance, common.FinalizerName)
		hasSynchronizationState := common.EventSkipped == instance.GetStatus().SynchronizationState
		annotationValue, annotationFound := instance.GetAnnotations()[annotations.SkipKey]
		hasAnnotationValue := annotationValue == strconv.FormatBool(true)

		return hasCorrelationID && hasFinalizer && hasSynchronizationState && annotationFound && hasAnnotationValue
	}, Timeout, Interval, "Client should not be processed")

	assert.NotEmpty(t, instance.GetStatus().CorrelationID)
	assert.Empty(t, instance.GetStatus().ClientID)
	assert.Empty(t, instance.GetStatus().KeyIDs)
	assert.Empty(t, instance.GetStatus().SynchronizationHash)
	assert.Empty(t, instance.GetStatus().SynchronizationTime)
	return instance
}
