package clients_test

import (
	"github.com/nais/digdirator/pkg/clients"
	"github.com/nais/digdirator/pkg/digdir/types"
	"github.com/nais/digdirator/pkg/secrets"
	"github.com/stretchr/testify/assert"
	"testing"
)

func TestGetIntegrationType(t *testing.T) {
	idPortenClient := minimalIDPortenClient()
	assert.Equal(t, types.IntegrationTypeIDPorten, clients.GetIntegrationType(idPortenClient))

	maskinportenClient := minimalMaskinportenClient()
	assert.Equal(t, types.IntegrationTypeMaskinporten, clients.GetIntegrationType(maskinportenClient))
}

func TestGetInstanceType(t *testing.T) {
	idPortenClient := minimalIDPortenClient()
	assert.Equal(t, "*nais_io_v1.IDPortenClient", clients.GetInstanceType(idPortenClient))

	maskinportenClient := minimalMaskinportenClient()
	assert.Equal(t, "*nais_io_v1.MaskinportenClient", clients.GetInstanceType(maskinportenClient))
}

func TestGetSecretName(t *testing.T) {
	idPortenClient := minimalIDPortenClient()
	idPortenClient.Spec.SecretName = "idporten-secret"
	assert.Equal(t, "idporten-secret", clients.GetSecretName(idPortenClient))

	maskinportenClient := minimalMaskinportenClient()
	maskinportenClient.Spec.SecretName = "maskinporten-secret"
	assert.Equal(t, "maskinporten-secret", clients.GetSecretName(maskinportenClient))
}

func TestGetSecretJwkKey(t *testing.T) {
	idPortenClient := minimalIDPortenClient()
	assert.Equal(t, secrets.IDPortenJwkKey, clients.GetSecretJwkKey(idPortenClient))

	maskinportenClient := minimalMaskinportenClient()
	assert.Equal(t, secrets.MaskinportenJwkKey, clients.GetSecretJwkKey(maskinportenClient))
}

func TestIsUpToDate(t *testing.T) {
	t.Run("Minimal IDPortenClient should be up-to-date", func(t *testing.T) {
		actual, err := clients.IsUpToDate(minimalIDPortenClient())
		assert.NoError(t, err)
		assert.True(t, actual)
	})

	t.Run("IDPortenClient with changed value should not be up-to-date", func(t *testing.T) {
		client := minimalIDPortenClient()
		client.Spec.ClientURI = "changed"
		actual, err := clients.IsUpToDate(client)
		assert.NoError(t, err)
		assert.False(t, actual)
	})

	t.Run("Minimal MaskinportenClient should be up-to-date", func(t *testing.T) {
		actual, err := clients.IsUpToDate(minimalMaskinportenClient())
		assert.NoError(t, err)
		assert.True(t, actual)
	})

	t.Run("MaskinportenClient with changed value should not be up-to-date", func(t *testing.T) {
		client := minimalMaskinportenClient()
		client.Spec.SecretName = "changed"
		actual, err := clients.IsUpToDate(client)
		assert.NoError(t, err)
		assert.False(t, actual)
	})

	t.Run("Minimal MaskinportenClientWithExternalInternal should be up-to-date", func(t *testing.T) {
		actual, err := clients.IsUpToDate(minimalMaskinportenWithScopeInternalExternalClient())
		assert.NoError(t, err)
		assert.True(t, actual)
	})

	t.Run("MaskinportenClientWithExternalInternal with changed value should not be up-to-date", func(t *testing.T) {
		client := minimalMaskinportenWithScopeInternalExternalClient()
		client.Spec.SecretName = "changed"
		actual, err := clients.IsUpToDate(client)
		assert.NoError(t, err)
		assert.False(t, actual)
	})
}

func TestToClientRegistration_IDPortenClient(t *testing.T) {
	client := minimalIDPortenClient()
	registration := clients.ToClientRegistration(client)

	assert.Equal(t, clients.IDPortenDefaultAccessTokenLifetimeSeconds, registration.AccessTokenLifetime)

	assert.Equal(t, types.ApplicationTypeWeb, registration.ApplicationType)

	assert.Equal(t, clients.IDPortenDefaultSessionLifetimeSeconds, registration.AuthorizationLifeTime)

	assert.Equal(t, clients.IDPortenDefaultClientURI, registration.ClientURI)

	assert.Equal(t, types.DefaultClientName, registration.ClientName)

	assert.Equal(t, "test-cluster:test-namespace:test-app", registration.Description)

	assert.True(t, registration.FrontchannelLogoutSessionRequired)
	assert.Empty(t, registration.FrontchannelLogoutURI)

	assert.Contains(t, registration.GrantTypes, types.GrantTypeAuthorizationCode)
	assert.Contains(t, registration.GrantTypes, types.GrantTypeRefreshToken)
	assert.Len(t, registration.GrantTypes, 2)

	assert.Equal(t, types.IntegrationTypeIDPorten, registration.IntegrationType)

	assert.Contains(t, registration.PostLogoutRedirectURIs, clients.IDPortenDefaultPostLogoutRedirectURI)
	assert.Len(t, registration.PostLogoutRedirectURIs, 1)

	assert.Contains(t, registration.RedirectURIs, "https://test.com")
	assert.Len(t, registration.RedirectURIs, 1)

	assert.Equal(t, clients.IDPortenDefaultSessionLifetimeSeconds, registration.RefreshTokenLifetime)

	assert.Equal(t, types.RefreshTokenUsageReuse, registration.RefreshTokenUsage)

	assert.Contains(t, registration.Scopes, "openid")
	assert.Contains(t, registration.Scopes, "profile")
	assert.Len(t, registration.Scopes, 2)

	assert.Equal(t, types.TokenEndpointAuthMethodPrivateKeyJwt, registration.TokenEndpointAuthMethod)
}

func TestToClientRegistration_MaskinportenClient(t *testing.T) {
	client := minimalMaskinportenClient()
	registration := clients.ToClientRegistration(client)

	assert.Equal(t, clients.IDPortenDefaultAccessTokenLifetimeSeconds, registration.AccessTokenLifetime)

	assert.Equal(t, types.ApplicationTypeWeb, registration.ApplicationType)

	assert.Equal(t, clients.IDPortenDefaultSessionLifetimeSeconds, registration.AuthorizationLifeTime)

	assert.Equal(t, clients.IDPortenDefaultClientURI, registration.ClientURI)

	assert.Equal(t, types.DefaultClientName, registration.ClientName)

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
