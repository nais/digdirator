package clients_test

import (
	"encoding/json"
	"testing"

	naisiov1 "github.com/nais/liberator/pkg/apis/nais.io/v1"
	"github.com/nais/liberator/pkg/oauth"
	"github.com/stretchr/testify/assert"

	"github.com/nais/digdirator/pkg/clients"
	"github.com/nais/digdirator/pkg/config"
	"github.com/nais/digdirator/pkg/crypto"
	"github.com/nais/digdirator/pkg/secrets"
)

func TestSecretData_IDPortenClient(t *testing.T) {
	client := minimalIDPortenClient()

	jwk, err := crypto.GenerateJwk()
	assert.NoError(t, err)

	cfg := makeConfig()

	stringData, err := clients.SecretData(client, *jwk, cfg)
	assert.NoError(t, err, "should not error")

	t.Run("StringData should contain expected fields and values", func(t *testing.T) {
		t.Run("Secret Data should contain "+secrets.IDPortenJwkKey, func(t *testing.T) {
			expected, err := json.Marshal(jwk)
			assert.NoError(t, err)
			assert.Equal(t, string(expected), stringData[secrets.IDPortenJwkKey])
		})
		t.Run("Secret Data should contain "+secrets.IDPortenWellKnownURLKey, func(t *testing.T) {
			expected := "https://idporten.example.com/.well-known/openid-configuration"
			assert.Equal(t, expected, stringData[secrets.IDPortenWellKnownURLKey])
		})
		t.Run("Secret Data should contain "+secrets.IDPortenClientIDKey, func(t *testing.T) {
			assert.Equal(t, client.Status.ClientID, stringData[secrets.IDPortenClientIDKey])
		})
		t.Run("Secret Data should contain "+secrets.IDPortenRedirectURIKey, func(t *testing.T) {
			assert.Equal(t, string(client.Spec.RedirectURI), stringData[secrets.IDPortenRedirectURIKey])
		})
	})
}

func TestSecretData_MaskinportenClient(t *testing.T) {
	client := minimalMaskinportenClient()
	client.Spec.Scopes = naisiov1.MaskinportenScope{
		ConsumedScopes: []naisiov1.ConsumedScope{
			{Name: "scope:one"},
			{Name: "scope:two"},
		},
	}

	jwk, err := crypto.GenerateJwk()
	assert.NoError(t, err)

	cfg := makeConfig()

	stringData, err := clients.SecretData(client, *jwk, cfg)
	assert.NoError(t, err, "should not error")

	t.Run("StringData should contain expected fields and values", func(t *testing.T) {
		t.Run("Secret Data should contain "+secrets.MaskinportenJwkKey, func(t *testing.T) {
			expected, err := json.Marshal(jwk)
			assert.NoError(t, err)
			assert.Equal(t, string(expected), stringData[secrets.MaskinportenJwkKey])
		})
		t.Run("Secret Data should contain "+secrets.MaskinportenWellKnownURLKey, func(t *testing.T) {
			expected := "https://maskinporten.example.com/.well-known/oauth-authorization-server"
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

func makeConfig() *config.Config {
	return &config.Config{
		DigDir: config.DigDir{
			IDPorten: config.IDPorten{
				WellKnownURL: "https://idporten.example.com/.well-known/openid-configuration",
				Metadata: oauth.MetadataOpenID{
					MetadataCommon: oauth.MetadataCommon{
						Issuer:        "https://idporten.example.com/",
						JwksURI:       "https://idporten.example.com/jwk",
						TokenEndpoint: "https://idporten.example.com/token",
					},
				},
			},
			Maskinporten: config.Maskinporten{
				WellKnownURL: "https://maskinporten.example.com/.well-known/oauth-authorization-server",
				Metadata: oauth.MetadataOAuth{
					MetadataCommon: oauth.MetadataCommon{
						Issuer:        "https://maskinporten.example.com/",
						JwksURI:       "https://maskinporten.example.com/jwk",
						TokenEndpoint: "https://maskinporten.example.com/token",
					},
				},
			},
		},
	}
}
