package v1

import (
	"github.com/nais/digdirator/pkg/digdir/types"
	"github.com/nais/digdirator/pkg/labels"
)

func (in *IDPortenClient) CalculateHash() (string, error) {
	return CalculateHash(in.Spec)
}

func (in *IDPortenClient) IsHashUnchanged() (bool, error) {
	return IsHashUnchanged(in)
}

func (in *IDPortenClient) GetIntegrationType() types.IntegrationType {
	return types.IntegrationTypeIDPorten
}

func (in *IDPortenClient) GetSecretName() string {
	return in.Spec.SecretName
}

func (in *IDPortenClient) GetStatus() *ClientStatus {
	return &in.Status
}

func (in *IDPortenClient) MakeLabels() map[string]string {
	return labels.IDPortenLabels(in)
}

func (in *IDPortenClient) MakeDescription() string {
	return MakeDescription(in)
}

func (in IDPortenClient) ToClientRegistration() types.ClientRegistration {
	return types.ClientRegistration{
		AccessTokenLifetime:               3600,
		ApplicationType:                   types.ApplicationTypeWeb,
		AuthorizationLifeTime:             0,
		ClientName:                        types.DefaultClientName,
		ClientURI:                         in.Spec.ClientURI,
		Description:                       in.MakeDescription(),
		FrontchannelLogoutSessionRequired: false,
		FrontchannelLogoutURI:             in.Spec.FrontchannelLogoutURI,
		GrantTypes: []types.GrantType{
			types.GrantTypeAuthorizationCode,
			types.GrantTypeRefreshToken,
		},
		IntegrationType:        types.IntegrationTypeIDPorten,
		PostLogoutRedirectURIs: in.Spec.PostLogoutRedirectURIs,
		RedirectURIs: []string{
			in.Spec.RedirectURI,
		},
		RefreshTokenLifetime: in.Spec.RefreshTokenLifetime,
		RefreshTokenUsage:    types.RefreshTokenUsageOneTime,
		Scopes: []string{
			"openid", "profile",
		},
		TokenEndpointAuthMethod: types.TokenEndpointAuthMethodPrivateKeyJwt,
	}
}
