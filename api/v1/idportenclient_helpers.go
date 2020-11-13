package v1

import (
	"fmt"
	"github.com/nais/digdirator/pkg/config"
	"github.com/nais/digdirator/pkg/digdir/types"
	"github.com/nais/digdirator/pkg/labels"
	"github.com/spf13/viper"
	"gopkg.in/square/go-jose.v2"
	"reflect"
)

func (in *IDPortenClient) CalculateHash() (string, error) {
	return calculateHash(in.Spec)
}

func (in *IDPortenClient) CreateSecretData(jwk jose.JSONWebKey) (map[string]string, error) {
	wellKnownURL := viper.GetString(config.DigDirIDPortenBaseURL) + "/idporten-oidc-provider/.well-known/openid-configuration"
	jwkJson, err := jwk.MarshalJSON()
	if err != nil {
		return nil, fmt.Errorf("marshalling JWK: %w", err)
	}
	return map[string]string{
		IDPortenJwkKey:       string(jwkJson),
		IDPortenWellKnownURL: wellKnownURL,
		IDPortenClientID:     in.GetStatus().GetClientID(),
		IDPortenRedirectURI:  in.Spec.RedirectURI,
	}, nil
}

func (in *IDPortenClient) GetIntegrationType() types.IntegrationType {
	return types.IntegrationTypeIDPorten
}

func (in *IDPortenClient) GetInstanceType() string {
	return reflect.TypeOf(in).String()
}

func (in *IDPortenClient) GetSecretMapKey() string {
	return IDPortenJwkKey
}

func (in *IDPortenClient) GetSecretName() string {
	return in.Spec.SecretName
}

func (in *IDPortenClient) GetStatus() *ClientStatus {
	return &in.Status
}

func (in *IDPortenClient) HasFinalizer(finalizerName string) bool {
	return hasFinalizer(in, finalizerName)
}

func (in *IDPortenClient) IsBeingDeleted() bool {
	return isBeingDeleted(in)
}

func (in *IDPortenClient) IsHashUnchanged() (bool, error) {
	return isHashUnchanged(in)
}

func (in *IDPortenClient) MakeLabels() map[string]string {
	return labels.IDPortenLabels(in)
}

func (in *IDPortenClient) MakeDescription() string {
	return makeDescription(in)
}

func (in IDPortenClient) ToClientRegistration() types.ClientRegistration {
	return types.ClientRegistration{
		AccessTokenLifetime:               3600,
		ApplicationType:                   types.ApplicationTypeWeb,
		AuthorizationLifeTime:             in.Spec.RefreshTokenLifetime,
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
		RefreshTokenUsage:    types.RefreshTokenUsageReuse,
		Scopes: []string{
			"openid", "profile",
		},
		TokenEndpointAuthMethod: types.TokenEndpointAuthMethodPrivateKeyJwt,
	}
}
