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

const expectedIDPortenClientHash = "d1c24e06524b718f"

func TestIDPortenClient_MakeDescription(t *testing.T) {
	expected := "test-cluster:test-namespace:test-app"
	assert.Equal(t, expected, minimalIDPortenClient().MakeDescription())
}

func TestIDPortenClient_CalculateHash(t *testing.T) {
	actual, err := minimalIDPortenClient().CalculateHash()
	assert.NoError(t, err)
	assert.Equal(t, expectedIDPortenClientHash, actual)
}

func TestIDPortenClient_MakeLabels(t *testing.T) {
	app := minimalIDPortenClient()
	expected := labels.IDPortenLabels(app)
	assert.Equal(t, app.MakeLabels(), expected)
}

func TestIDPortenClient_IsUpToDate(t *testing.T) {
	t.Run("Minimal Application should be up-to-date", func(t *testing.T) {
		actual, err := minimalIDPortenClient().IsUpToDate()
		assert.NoError(t, err)
		assert.True(t, actual)
	})
	t.Run("Application with changed value should not be up-to-date", func(t *testing.T) {
		app := minimalIDPortenClient()
		app.Spec.ClientURI = "changed"
		actual, err := app.IsUpToDate()
		assert.NoError(t, err)
		assert.False(t, actual)
	})
}

func TestIDPortenClient_SetHash(t *testing.T) {
	app := minimalIDPortenClient()
	app.Spec.ClientURI = "changed"

	hash, err := app.CalculateHash()
	assert.NoError(t, err)
	app.GetStatus().SetHash(hash)
	assert.Equal(t, "95d4f91a6b5cb768", app.GetStatus().GetSynchronizationHash())
}

func TestIDPortenClient_IntegrationType(t *testing.T) {
	app := minimalIDPortenClient()
	assert.Equal(t, types.IntegrationTypeIDPorten, app.GetIntegrationType())
}

func TestIDPortenClient_CreateSecretData(t *testing.T) {
	client := minimalIDPortenClient()

	jwk, err := crypto.GenerateJwk()
	assert.NoError(t, err)

	stringData, err := client.CreateSecretData(*jwk)
	assert.NoError(t, err, "should not error")

	t.Run("StringData should contain expected fields and values", func(t *testing.T) {
		t.Run("Secret Data should contain "+v1.IDPortenJwkKey, func(t *testing.T) {
			expected, err := json.Marshal(jwk)
			assert.NoError(t, err)
			assert.Equal(t, string(expected), stringData[v1.IDPortenJwkKey])
		})
		t.Run("Secret Data should contain "+v1.IDPortenWellKnownURL, func(t *testing.T) {
			expected := viper.GetString(config.DigDirIDPortenBaseURL) + "/idporten-oidc-provider/.well-known/openid-configuration"
			assert.Equal(t, expected, stringData[v1.IDPortenWellKnownURL])
		})
		t.Run("Secret Data should contain "+v1.IDPortenClientID, func(t *testing.T) {
			assert.Equal(t, client.Status.ClientID, stringData[v1.IDPortenClientID])
		})
		t.Run("Secret Data should contain "+v1.IDPortenRedirectURI, func(t *testing.T) {
			assert.Equal(t, client.Spec.RedirectURI, stringData[v1.IDPortenRedirectURI])
		})
	})
}

func TestIDPortenClient_GetSecretMapKey(t *testing.T) {
	client := minimalIDPortenClient()
	assert.Equal(t, v1.IDPortenJwkKey, client.GetSecretMapKey())
}

func minimalIDPortenClient() *v1.IDPortenClient {
	return &v1.IDPortenClient{
		ObjectMeta: metav1.ObjectMeta{
			Name:        "test-app",
			Namespace:   "test-namespace",
			ClusterName: "test-cluster",
		},
		Spec: v1.IDPortenClientSpec{
			ClientURI:              "test",
			RedirectURI:            "https://test.com",
			SecretName:             "test",
			FrontchannelLogoutURI:  "test",
			PostLogoutRedirectURIs: nil,
		},
		Status: v1.ClientStatus{
			SynchronizationHash:  expectedIDPortenClientHash,
			SynchronizationState: v1.EventSynchronized,
		},
	}
}
