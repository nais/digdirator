package v1

import (
	"testing"

	"github.com/stretchr/testify/assert"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func TestMaskinportenClient_GetUniqueName(t *testing.T) {
	expected := "maskinporten:test-cluster:test-namespace:test-app"
	assert.Equal(t, expected, minimalMaskinportenClient().GetUniqueName())
}

func TestMaskinportenClient_HasFinalizer(t *testing.T) {
	t.Run("Minimal Application should not have finalizer", func(t *testing.T) {
		assert.False(t, minimalClient().HasFinalizer(finalizerName))
	})
	t.Run("Application with finalizer should have finalizer", func(t *testing.T) {
		app := minimalClient()
		app.ObjectMeta.Finalizers = []string{finalizerName}
		assert.True(t, app.HasFinalizer(finalizerName))
	})
}

func TestMaskinportenClient_AddFinalizer(t *testing.T) {
	app := minimalClient()
	t.Run("Minimal Application should not have finalizer", func(t *testing.T) {
		assert.False(t, app.HasFinalizer(finalizerName))
	})
	t.Run("Application should have finalizer after add", func(t *testing.T) {
		app.AddFinalizer(finalizerName)
		assert.True(t, app.HasFinalizer(finalizerName))
	})
}

func TestMaskinporten_RemoveFinalizer(t *testing.T) {
	app := minimalClient()
	app.ObjectMeta.Finalizers = []string{finalizerName}
	t.Run("Minimal Application should have finalizer", func(t *testing.T) {
		assert.True(t, app.HasFinalizer(finalizerName))
	})
	t.Run("Application should not have finalizer after remove", func(t *testing.T) {
		app.RemoveFinalizer(finalizerName)
		actual := app.HasFinalizer(finalizerName)
		assert.False(t, actual)
	})
}

func TestMaskinporten_IsBeingDeleted(t *testing.T) {
	t.Run("Minimal Application without deletion marker should not be marked for deletion", func(t *testing.T) {
		assert.False(t, minimalClient().IsBeingDeleted())
	})
	t.Run("Application with deletion marker should be marked for deletion", func(t *testing.T) {
		app := minimalClient()
		now := metav1.Now()
		app.ObjectMeta.DeletionTimestamp = &now
		assert.True(t, app.IsBeingDeleted())
	})
}

func TestMaskinportenClient_Hash(t *testing.T) {
	actual, err := minimalClient().Hash()
	assert.NoError(t, err)
	assert.Equal(t, expectedHash, actual)
}

func TestMaskinportenClient_HashUnchanged(t *testing.T) {
	t.Run("Minimal Application should have unchanged hash value", func(t *testing.T) {
		actual, err := minimalClient().HashUnchanged()
		assert.NoError(t, err)
		assert.True(t, actual)
	})
	t.Run("Application with changed value should have changed hash value", func(t *testing.T) {
		app := minimalClient()
		app.Spec.ClientURI = "changed"
		actual, err := app.HashUnchanged()
		assert.NoError(t, err)
		assert.False(t, actual)
	})
}

func TestMaskinportenClient_UpdateHash(t *testing.T) {
	app := minimalMaskinportenClient()
	app.Spec.Scopes = []string{"some:another/scope"}

	err := app.UpdateHash()
	assert.NoError(t, err)
	assert.Equal(t, "db68f38423a12192", app.Status.ProvisionHash)
}

func minimalMaskinportenClient() *MaskinportenClient {
	return &MaskinportenClient{
		ObjectMeta: metav1.ObjectMeta{
			Name:        "test-app",
			Namespace:   "test-namespace",
			ClusterName: "test-cluster",
		},
		Spec: MaskinportenClientSpec{
			Scopes: nil,
		},
		Status: MaskinportenClientStatus{
			ProvisionHash: expectedHash,
		},
	}
}
