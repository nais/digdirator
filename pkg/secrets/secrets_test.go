package secrets_test

import (
	"encoding/json"
	"testing"

	naisiov1 "github.com/nais/liberator/pkg/apis/nais.io/v1"
	"github.com/nais/liberator/pkg/oauth"
	"github.com/stretchr/testify/assert"

	"github.com/nais/digdirator/pkg/config"
	"github.com/nais/digdirator/pkg/crypto"
	"github.com/nais/digdirator/pkg/fixtures"
	"github.com/nais/digdirator/pkg/secrets"
)

func TestIDPortenClientSecretData(t *testing.T) {
	client := fixtures.MinimalIDPortenClient()

	jwk, err := crypto.GenerateJwk()
	assert.NoError(t, err)

	cfg := makeConfig()

	stringData, err := secrets.IDPortenClientSecretData(client, *jwk, cfg)
	assert.NoError(t, err, "should not error")

	t.Run("StringData should contain expected fields and values", func(t *testing.T) {
		t.Run("Secret Data should contain "+secrets.IDPortenJwkKey, func(t *testing.T) {
			expected, err := json.Marshal(jwk)
			assert.NoError(t, err)
			assert.Equal(t, string(expected), stringData[secrets.IDPortenJwkKey])
		})

		for _, test := range []struct {
			key      string
			expected string
		}{
			{
				key:      secrets.IDPortenClientIDKey,
				expected: "test-idporten",
			},
			{
				key:      secrets.IDPortenWellKnownURLKey,
				expected: "https://idporten.example.com/.well-known/openid-configuration",
			},
			{
				key:      secrets.IDPortenIssuerKey,
				expected: "https://idporten.example.com/",
			},
			{
				key:      secrets.IDPortenJwksUriKey,
				expected: "https://idporten.example.com/jwk",
			},
			{
				key:      secrets.IDPortenTokenEndpointKey,
				expected: "https://idporten.example.com/token",
			},
		} {
			t.Run("Secret Data should contain "+test.key, func(t *testing.T) {
				assert.NotEmpty(t, stringData[test.key])
				assert.Equal(t, test.expected, stringData[test.key])
			})
		}
	})
}

func TestMaskinportenClientSecretData(t *testing.T) {
	client := fixtures.MinimalMaskinportenClient()
	client.Spec.Scopes = naisiov1.MaskinportenScope{
		ConsumedScopes: []naisiov1.ConsumedScope{
			{Name: "scope:one"},
			{Name: "scope:two"},
		},
	}

	jwk, err := crypto.GenerateJwk()
	assert.NoError(t, err)

	cfg := makeConfig()

	stringData, err := secrets.MaskinportenClientSecretData(client, *jwk, cfg)
	assert.NoError(t, err, "should not error")

	t.Run("StringData should contain expected fields and values", func(t *testing.T) {
		t.Run("Secret Data should contain "+secrets.MaskinportenJwkKey, func(t *testing.T) {
			expected, err := json.Marshal(jwk)
			assert.NoError(t, err)
			assert.Equal(t, string(expected), stringData[secrets.MaskinportenJwkKey])
		})

		for _, test := range []struct {
			key      string
			expected string
		}{
			{
				key:      secrets.MaskinportenClientIDKey,
				expected: "test-maskinporten",
			},
			{
				key:      secrets.MaskinportenWellKnownURLKey,
				expected: "https://maskinporten.example.com/.well-known/oauth-authorization-server",
			},
			{
				key:      secrets.MaskinportenIssuerKey,
				expected: "https://maskinporten.example.com/",
			},
			{
				key:      secrets.MaskinportenJwksUriKey,
				expected: "https://maskinporten.example.com/jwk",
			},
			{
				key:      secrets.MaskinportenTokenEndpointKey,
				expected: "https://maskinporten.example.com/token",
			},
			{
				key:      secrets.MaskinportenScopesKey,
				expected: "scope:one scope:two",
			},
		} {
			t.Run("Secret Data should contain "+test.key, func(t *testing.T) {
				assert.NotEmpty(t, stringData[test.key])
				assert.Equal(t, test.expected, stringData[test.key])
			})
		}
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
