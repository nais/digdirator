package v1

import (
	"github.com/nais/digdirator/pkg/digdir/types"
	"github.com/nais/digdirator/pkg/labels"
)

func (in *MaskinportenClient) CalculateHash() (string, error) {
	return CalculateHash(in.Spec)
}

func (in *MaskinportenClient) IsHashUnchanged() (bool, error) {
	return IsHashUnchanged(in)
}

func (in *MaskinportenClient) GetIntegrationType() types.IntegrationType {
	return types.IntegrationTypeMaskinporten
}

func (in *MaskinportenClient) GetSecretName() string {
	return in.Spec.SecretName
}

func (in *MaskinportenClient) GetStatus() *ClientStatus {
	return &in.Status
}

func (in *MaskinportenClient) MakeLabels() map[string]string {
	return labels.MaskinportenLabels(in)
}

func (in *MaskinportenClient) MakeDescription() string {
	return MakeDescription(in)
}

func (in MaskinportenClient) ToClientRegistration() types.ClientRegistration {
	return types.ClientRegistration{
		AccessTokenLifetime:               3600,
		ApplicationType:                   types.ApplicationTypeWeb,
		AuthorizationLifeTime:             0,
		ClientName:                        types.DefaultClientName,
		ClientURI:                         "",
		Description:                       in.MakeDescription(),
		FrontchannelLogoutSessionRequired: false,
		FrontchannelLogoutURI:             "",
		GrantTypes: []types.GrantType{
			types.GrantTypeJwtBearer,
		},
		IntegrationType:         types.IntegrationTypeMaskinporten,
		PostLogoutRedirectURIs:  nil,
		RedirectURIs:            nil,
		RefreshTokenLifetime:    0,
		RefreshTokenUsage:       types.RefreshTokenUsageOneTime,
		Scopes:                  in.Spec.Scopes,
		TokenEndpointAuthMethod: types.TokenEndpointAuthMethodPrivateKeyJwt,
	}
}
