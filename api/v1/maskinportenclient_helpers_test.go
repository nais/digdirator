package v1

import (
	"github.com/nais/digdirator/pkg/digdir/types"
	"github.com/nais/digdirator/pkg/labels"
	"testing"

	"github.com/stretchr/testify/assert"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

const expectedMaskinportenClientHash = "98105fd6e1607430"

func TestMaskinportenClient_MakeDescription(t *testing.T) {
	expected := "test-cluster:test-namespace:test-app"
	assert.Equal(t, expected, minimalMaskinportenClient().MakeDescription())
}

func TestMaskinportenClient_CalculateHash(t *testing.T) {
	actual, err := minimalMaskinportenClient().CalculateHash()
	assert.NoError(t, err)
	assert.Equal(t, expectedMaskinportenClientHash, actual)
}

func TestMaskinportenClient_MakeLabels(t *testing.T) {
	app := minimalMaskinportenClient()
	expected := labels.MaskinportenLabels(app)
	assert.Equal(t, app.MakeLabels(), expected)
}

func TestMaskinportenClient_IsHashUnchanged(t *testing.T) {
	t.Run("Minimal Application should have unchanged hash value", func(t *testing.T) {
		actual, err := minimalMaskinportenClient().IsHashUnchanged()
		assert.NoError(t, err)
		assert.True(t, actual)
	})
	t.Run("Application with changed value should have changed hash value", func(t *testing.T) {
		app := minimalMaskinportenClient()
		app.Spec.SecretName = "changed"
		actual, err := app.IsHashUnchanged()
		assert.NoError(t, err)
		assert.False(t, actual)
	})
}

func TestMaskinportenClient_SetHash(t *testing.T) {
	app := minimalMaskinportenClient()
	app.Spec.Scopes = []string{"some:another/scope"}

	hash, err := app.CalculateHash()
	assert.NoError(t, err)
	app.GetStatus().SetHash(hash)
	assert.Equal(t, "ddbcd7f2b2711184", app.GetStatus().GetHash())
}

func TestMaskinportenClient_GetIntegrationType(t *testing.T) {
	app := minimalMaskinportenClient()
	assert.Equal(t, types.IntegrationTypeMaskinporten, app.GetIntegrationType())
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
			ProvisionHash: expectedMaskinportenClientHash,
		},
	}
}
