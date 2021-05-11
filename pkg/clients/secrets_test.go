package clients_test

import (
	"encoding/json"
	"github.com/nais/digdirator/pkg/clients"
	"github.com/nais/digdirator/pkg/config"
	"github.com/nais/digdirator/pkg/crypto"
	"github.com/nais/digdirator/pkg/secrets"
	naisiov1 "github.com/nais/liberator/pkg/apis/nais.io/v1"
	"github.com/spf13/viper"
	"github.com/stretchr/testify/assert"
	"testing"
)

func TestSecretData_IDPortenClient(t *testing.T) {
	client := minimalIDPortenClient()

	jwk, err := crypto.GenerateJwk()
	assert.NoError(t, err)

	stringData, err := clients.SecretData(client, *jwk)
	assert.NoError(t, err, "should not error")

	t.Run("StringData should contain expected fields and values", func(t *testing.T) {
		t.Run("Secret Data should contain "+secrets.IDPortenJwkKey, func(t *testing.T) {
			expected, err := json.Marshal(jwk)
			assert.NoError(t, err)
			assert.Equal(t, string(expected), stringData[secrets.IDPortenJwkKey])
		})
		t.Run("Secret Data should contain "+secrets.IDPortenWellKnownURLKey, func(t *testing.T) {
			expected := viper.GetString(config.DigDirIDPortenBaseURL) + "/idporten-oidc-provider/.well-known/openid-configuration"
			assert.Equal(t, expected, stringData[secrets.IDPortenWellKnownURLKey])
		})
		t.Run("Secret Data should contain "+secrets.IDPortenClientIDKey, func(t *testing.T) {
			assert.Equal(t, client.Status.ClientID, stringData[secrets.IDPortenClientIDKey])
		})
		t.Run("Secret Data should contain "+secrets.IDPortenRedirectURIKey, func(t *testing.T) {
			assert.Equal(t, client.Spec.RedirectURI, stringData[secrets.IDPortenRedirectURIKey])
		})
	})
}

func TestSecretData_MaskinportenClient(t *testing.T) {
	client := minimalMaskinportenClient()
	client.Spec.Scopes = naisiov1.MaskinportenScope{
		UsedScope: []naisiov1.UsedScope{
			{Name: "scope:one"},
			{Name: "scope:two"},
		},
	}

	jwk, err := crypto.GenerateJwk()
	assert.NoError(t, err)

	stringData, err := clients.SecretData(client, *jwk)
	assert.NoError(t, err, "should not error")

	t.Run("StringData should contain expected fields and values", func(t *testing.T) {
		t.Run("Secret Data should contain "+secrets.MaskinportenJwkKey, func(t *testing.T) {
			expected, err := json.Marshal(jwk)
			assert.NoError(t, err)
			assert.Equal(t, string(expected), stringData[secrets.MaskinportenJwkKey])
		})
		t.Run("Secret Data should contain "+secrets.MaskinportenWellKnownURLKey, func(t *testing.T) {
			expected := viper.GetString(config.DigDirMaskinportenBaseURL) + "/.well-known/oauth-authorization-server"
			assert.Equal(t, expected, stringData[secrets.MaskinportenWellKnownURLKey])
		})
		t.Run("Secret Data should contain "+secrets.MaskinportenClientIDKey, func(t *testing.T) {
			assert.Equal(t, client.Status.ClientID, stringData[secrets.MaskinportenClientIDKey])
		})
		t.Run("Secret Data should contain "+secrets.MaskinportenScopesKey+" with a single string of scopes separated by space", func(t *testing.T) {
			assert.Equal(t, "scope:one scope:two", stringData[secrets.MaskinportenScopesKey])
		})
	})
}
