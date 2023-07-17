package clients_test

import (
	"testing"

	naisiov1 "github.com/nais/liberator/pkg/apis/nais.io/v1"
	"github.com/stretchr/testify/assert"

	"github.com/nais/digdirator/pkg/clients"
	"github.com/nais/digdirator/pkg/config"
	"github.com/nais/digdirator/pkg/digdir/types"
	"github.com/nais/digdirator/pkg/fixtures"
	"github.com/nais/digdirator/pkg/secrets"
)

func TestGetIntegrationType(t *testing.T) {
	idPortenClient := fixtures.MinimalIDPortenClient()
	assert.Equal(t, types.IntegrationTypeIDPorten, clients.GetIntegrationType(idPortenClient))

	idPortenClient.Spec.IntegrationType = string(types.IntegrationTypeApiKlient)
	assert.Equal(t, types.IntegrationTypeApiKlient, clients.GetIntegrationType(idPortenClient))

	maskinportenClient := fixtures.MinimalMaskinportenClient()
	assert.Equal(t, types.IntegrationTypeMaskinporten, clients.GetIntegrationType(maskinportenClient))
}

func TestGetInstanceType(t *testing.T) {
	idPortenClient := fixtures.MinimalIDPortenClient()
	assert.Equal(t, "*nais_io_v1.IDPortenClient", clients.GetInstanceType(idPortenClient))

	maskinportenClient := fixtures.MinimalMaskinportenClient()
	assert.Equal(t, "*nais_io_v1.MaskinportenClient", clients.GetInstanceType(maskinportenClient))
}

func TestGetSecretName(t *testing.T) {
	idPortenClient := fixtures.MinimalIDPortenClient()
	idPortenClient.Spec.SecretName = "idporten-secret"
	assert.Equal(t, "idporten-secret", clients.GetSecretName(idPortenClient))

	maskinportenClient := fixtures.MinimalMaskinportenClient()
	maskinportenClient.Spec.SecretName = "maskinporten-secret"
	assert.Equal(t, "maskinporten-secret", clients.GetSecretName(maskinportenClient))
}

func TestGetSecretJwkKey(t *testing.T) {
	idPortenClient := fixtures.MinimalIDPortenClient()
	assert.Equal(t, secrets.IDPortenJwkKey, clients.GetSecretJwkKey(idPortenClient))

	maskinportenClient := fixtures.MinimalMaskinportenClient()
	assert.Equal(t, secrets.MaskinportenJwkKey, clients.GetSecretJwkKey(maskinportenClient))
}

func TestIsUpToDate(t *testing.T) {
	t.Run("Minimal IDPortenClient should be up-to-date", func(t *testing.T) {
		actual, err := clients.IsUpToDate(fixtures.MinimalIDPortenClient())
		assert.NoError(t, err)
		assert.True(t, actual)
	})

	t.Run("IDPortenClient with changed value should not be up-to-date", func(t *testing.T) {
		client := fixtures.MinimalIDPortenClient()
		client.Spec.ClientURI = "changed"
		actual, err := clients.IsUpToDate(client)
		assert.NoError(t, err)
		assert.False(t, actual)
	})

	t.Run("Minimal MaskinportenClient should be up-to-date", func(t *testing.T) {
		actual, err := clients.IsUpToDate(fixtures.MinimalMaskinportenClient())
		assert.NoError(t, err)
		assert.True(t, actual)
	})

	t.Run("MaskinportenClient with changed value should not be up-to-date", func(t *testing.T) {
		client := fixtures.MinimalMaskinportenClient()
		client.Spec.SecretName = "changed"
		actual, err := clients.IsUpToDate(client)
		assert.NoError(t, err)
		assert.False(t, actual)
	})

	t.Run("Minimal MaskinportenClientWithExternalInternal should be up-to-date", func(t *testing.T) {
		actual, err := clients.IsUpToDate(fixtures.MinimalMaskinportenWithScopeInternalExposedClient())
		assert.NoError(t, err)
		assert.True(t, actual)
	})

	t.Run("MaskinportenClientWithExternalInternal with changed value should not be up-to-date", func(t *testing.T) {
		client := fixtures.MinimalMaskinportenWithScopeInternalExposedClient()
		client.Spec.SecretName = "changed"
		actual, err := clients.IsUpToDate(client)
		assert.NoError(t, err)
		assert.False(t, actual)
	})
}

func TestToClientRegistration_IDPortenClient(t *testing.T) {
	client := fixtures.MinimalIDPortenClient()
	cluster := "test-cluster"
	cfg := makeConfig(cluster)
	registration := clients.ToClientRegistration(client, cfg)

	assert.Equal(t, 3600, registration.AccessTokenLifetime)

	assert.Equal(t, types.ApplicationTypeWeb, registration.ApplicationType)

	assert.Equal(t, 7200, registration.AuthorizationLifeTime)

	assert.Equal(t, "https://some-client-uri", registration.ClientURI)

	assert.Equal(t, "some-client-name", registration.ClientName)

	assert.Equal(t, "test-cluster:test-namespace:test-app", registration.Description)

	assert.True(t, registration.FrontchannelLogoutSessionRequired)
	assert.Empty(t, registration.FrontchannelLogoutURI)

	assert.Contains(t, registration.GrantTypes, types.GrantTypeAuthorizationCode)
	assert.Contains(t, registration.GrantTypes, types.GrantTypeRefreshToken)
	assert.Len(t, registration.GrantTypes, 2)

	assert.Equal(t, types.IntegrationTypeIDPorten, registration.IntegrationType)

	assert.Contains(t, registration.PostLogoutRedirectURIs, "https://some-client-uri")
	assert.Len(t, registration.PostLogoutRedirectURIs, 1)

	assert.Contains(t, registration.RedirectURIs, "https://test.com")
	assert.Len(t, registration.RedirectURIs, 1)

	assert.Equal(t, 7200, registration.RefreshTokenLifetime)

	assert.Equal(t, types.RefreshTokenUsageReuse, registration.RefreshTokenUsage)

	assert.Contains(t, registration.Scopes, "openid")
	assert.Contains(t, registration.Scopes, "profile")
	assert.Len(t, registration.Scopes, 2)

	assert.Equal(t, types.TokenEndpointAuthMethodPrivateKeyJwt, registration.TokenEndpointAuthMethod)

	t.Run("deprecated redirectURI field is preserved in the registration payload", func(t *testing.T) {
		client.Spec.RedirectURI = "https://test.com"
		client.Spec.RedirectURIs = []naisiov1.IDPortenURI{
			"https://test.com/a",
			"https://test.com/b",
			"https://test.com/b",
		}
		registration = clients.ToClientRegistration(client, cfg)
		assert.ElementsMatch(t, registration.RedirectURIs, []string{
			"https://test.com",
			"https://test.com/a",
			"https://test.com/b",
		})
		assert.Len(t, registration.RedirectURIs, 3)
	})

	t.Run("integration type maskinporten should not be allowed", func(t *testing.T) {
		client.Spec.IntegrationType = string(types.IntegrationTypeMaskinporten)
		registration = clients.ToClientRegistration(client, cfg)
		assert.Equal(t, types.IntegrationTypeIDPorten, registration.IntegrationType)
	})
}

func TestToClientRegistration_MaskinportenClient(t *testing.T) {
	client := fixtures.MinimalMaskinportenClient()
	cluster := "test-cluster"
	cfg := makeConfig(cluster)
	registration := clients.ToClientRegistration(client, cfg)

	assert.Equal(t, 3600, registration.AccessTokenLifetime)

	assert.Equal(t, types.ApplicationTypeWeb, registration.ApplicationType)

	assert.Equal(t, 7200, registration.AuthorizationLifeTime)

	assert.Equal(t, "https://some-client-uri", registration.ClientURI)

	assert.Equal(t, "some-client-name", registration.ClientName)

	assert.Equal(t, "test-cluster:test-namespace:test-app", registration.Description)

	assert.False(t, registration.FrontchannelLogoutSessionRequired)
	assert.Empty(t, registration.FrontchannelLogoutURI)

	assert.Contains(t, registration.GrantTypes, types.GrantTypeJwtBearer)
	assert.Len(t, registration.GrantTypes, 1)

	assert.Equal(t, types.IntegrationTypeMaskinporten, registration.IntegrationType)

	assert.Nil(t, registration.PostLogoutRedirectURIs)

	assert.Nil(t, registration.RedirectURIs)

	assert.Equal(t, 0, registration.RefreshTokenLifetime)

	assert.Equal(t, types.RefreshTokenUsageOneTime, registration.RefreshTokenUsage)

	assert.Contains(t, registration.Scopes, "some-scope")
	assert.Len(t, registration.Scopes, 1)

	assert.Equal(t, types.TokenEndpointAuthMethodPrivateKeyJwt, registration.TokenEndpointAuthMethod)
}

func TestToClientRegistration_IntegrationType(t *testing.T) {
	client := fixtures.MinimalIDPortenClient()
	cluster := "test-cluster"
	cfg := makeConfig(cluster)

	for _, test := range []struct {
		name                     string
		specifiedIntegrationType string
		wantIntegrationType      types.IntegrationType
		wantScopes               []string
	}{
		{
			name:                "No integrationType specified then return default IntegrationType and scope",
			wantIntegrationType: types.IntegrationTypeIDPorten,
			wantScopes:          []string{"openid", "profile"},
		},
		{
			name:                     "IntegrationType idporten specified then return specified IntegrationType and and matching scope",
			specifiedIntegrationType: string(types.IntegrationTypeIDPorten),
			wantIntegrationType:      types.IntegrationTypeIDPorten,
			wantScopes:               []string{"openid", "profile"},
		},
		{
			name:                     "IntegrationType api_klient specified then return specified IntegrationType and and matching scope (if any)",
			specifiedIntegrationType: string(types.IntegrationTypeApiKlient),
			wantIntegrationType:      types.IntegrationTypeApiKlient,
		},
		{
			name:                     "IntegrationType krr specified then return specified IntegrationType and and matching scope",
			specifiedIntegrationType: string(types.IntegrationTypeKrr),
			wantIntegrationType:      types.IntegrationTypeKrr,
			wantScopes:               []string{"krr:global/kontaktinformasjon.read", "krr:global/digitalpost.read"},
		},
		{
			name:                     "Unknown IntegrationType (should not happen)",
			specifiedIntegrationType: string(types.IntegrationTypeUnknown),
			wantIntegrationType:      types.IntegrationTypeUnknown,
		},
	} {
		if len(test.specifiedIntegrationType) > 0 {
			client.Spec.IntegrationType = test.specifiedIntegrationType
		}

		actual := clients.ToClientRegistration(client, cfg)

		t.Run(test.name, func(t *testing.T) {
			assert.Equal(t, test.wantIntegrationType, actual.IntegrationType)
			assert.Equal(t, test.wantScopes, actual.Scopes)
		})
	}
}

func TestIDPortenIntegrationNameFallback(t *testing.T) {
	client := fixtures.MinimalIDPortenClient()
	cluster := "test-cluster"

	cfg1 := makeConfig(cluster)
	registration1 := clients.ToClientRegistration(client, cfg1)
	assert.Equal(t, cfg1.DigDir.Common.ClientName, registration1.ClientName)

	integrationName := "test-integration"
	client.Spec.IntegrationName = integrationName
	cfg2 := makeConfig(cluster)
	registration2 := clients.ToClientRegistration(client, cfg2)
	assert.Equal(t, integrationName, registration2.ClientName)
}

func TestMaskinportenIntegrationNameFallback(t *testing.T) {
	client := fixtures.MinimalMaskinportenClient()
	cluster := "test-cluster"

	cfg1 := makeConfig(cluster)
	registration1 := clients.ToClientRegistration(client, cfg1)
	assert.Equal(t, cfg1.DigDir.Common.ClientName, registration1.ClientName)

	integrationName := "test-integration"
	client.Spec.IntegrationName = integrationName
	cfg2 := makeConfig(cluster)
	registration2 := clients.ToClientRegistration(client, cfg2)
	assert.Equal(t, integrationName, registration2.ClientName)
}

func makeConfig(clusterName string) *config.Config {
	return &config.Config{
		ClusterName: clusterName,
		DigDir: config.DigDir{
			Common: config.DigDirCommon{
				AccessTokenLifetime: 3600,
				ClientName:          "some-client-name",
				ClientURI:           "https://some-client-uri",
				SessionLifetime:     7200,
			},
		},
	}
}
