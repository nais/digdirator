package v1_test

import (
	"encoding/json"
	v1 "github.com/nais/digdirator/api/v1"
	"github.com/nais/digdirator/pkg/config"
	"github.com/nais/digdirator/pkg/crypto"
	"github.com/nais/digdirator/pkg/digdir/types"
	"github.com/nais/digdirator/pkg/labels"
	"github.com/spf13/viper"
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

func TestMaskinportenClient_IsUpToDate(t *testing.T) {
	t.Run("Minimal Application should be up-to-date", func(t *testing.T) {
		actual, err := minimalMaskinportenClient().IsUpToDate()
		assert.NoError(t, err)
		assert.True(t, actual)
	})
	t.Run("Application with changed value should not be up-to-date", func(t *testing.T) {
		app := minimalMaskinportenClient()
		app.Spec.SecretName = "changed"
		actual, err := app.IsUpToDate()
		assert.NoError(t, err)
		assert.False(t, actual)
	})
}

func TestMaskinportenClient_SetHash(t *testing.T) {
	app := minimalMaskinportenClient()
	app.Spec.Scopes = []v1.MaskinportenScope{
		{Name: "some:another/scope"},
	}

	hash, err := app.CalculateHash()
	assert.NoError(t, err)
	app.GetStatus().SetHash(hash)
	assert.Equal(t, "7a8e489f74dda6d7", app.GetStatus().GetSynchronizationHash())
}

func TestMaskinportenClient_GetIntegrationType(t *testing.T) {
	app := minimalMaskinportenClient()
	assert.Equal(t, types.IntegrationTypeMaskinporten, app.GetIntegrationType())
}

func TestMaskinporten_CreateSecretData(t *testing.T) {
	client := minimalMaskinportenClient()
	client.Spec.Scopes = []v1.MaskinportenScope{
		{Name: "scope:one"},
		{Name: "scope:two"},
	}

	jwk, err := crypto.GenerateJwk()
	assert.NoError(t, err)

	stringData, err := client.CreateSecretData(*jwk)
	assert.NoError(t, err, "should not error")

	t.Run("StringData should contain expected fields and values", func(t *testing.T) {
		t.Run("Secret Data should contain "+v1.MaskinportenJwkKey, func(t *testing.T) {
			expected, err := json.Marshal(jwk)
			assert.NoError(t, err)
			assert.Equal(t, string(expected), stringData[v1.MaskinportenJwkKey])
		})
		t.Run("Secret Data should contain "+v1.MaskinportenWellKnownURL, func(t *testing.T) {
			expected := viper.GetString(config.DigDirMaskinportenBaseURL) + "/.well-known/oauth-authorization-server"
			assert.Equal(t, expected, stringData[v1.MaskinportenWellKnownURL])
		})
		t.Run("Secret Data should contain "+v1.MaskinportenClientID, func(t *testing.T) {
			assert.Equal(t, client.Status.ClientID, stringData[v1.MaskinportenClientID])
		})
		t.Run("Secret Data should contain "+v1.MaskinportenScopes+" with a single string of scopes separated by space", func(t *testing.T) {
			assert.Equal(t, "scope:one scope:two", stringData[v1.MaskinportenScopes])
		})
	})
}

func TestMaskinportenClient_GetSecretMapKey(t *testing.T) {
	client := minimalMaskinportenClient()
	assert.Equal(t, v1.MaskinportenJwkKey, client.GetSecretMapKey())
}

func minimalMaskinportenClient() *v1.MaskinportenClient {
	return &v1.MaskinportenClient{
		ObjectMeta: metav1.ObjectMeta{
			Name:        "test-app",
			Namespace:   "test-namespace",
			ClusterName: "test-cluster",
		},
		Spec: v1.MaskinportenClientSpec{
			Scopes: nil,
		},
		Status: v1.ClientStatus{
			SynchronizationHash:  expectedMaskinportenClientHash,
			SynchronizationState: v1.EventSynchronized,
		},
	}
}
