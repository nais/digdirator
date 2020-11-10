package v1

import (
	"github.com/nais/digdirator/pkg/digdir/types"
	"testing"

	"github.com/stretchr/testify/assert"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func TestMaskinportenClient_GetUniqueName(t *testing.T) {
	expected := "test-cluster:test-namespace:test-app"
	assert.Equal(t, expected, minimalMaskinportenClient().GetUniqueName())
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
	assert.Equal(t, "ddbcd7f2b2711184", app.Status.ProvisionHash)
}

func TestMaskinportenClient_IntegrationType(t *testing.T) {
	app := minimalMaskinportenClient()
	assert.Equal(t, types.IntegrationTypeMaskinporten, app.IntegrationType())
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
		Status: ClientStatus{
			ProvisionHash: expectedHash,
		},
	}
}
