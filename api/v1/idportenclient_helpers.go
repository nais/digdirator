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

func (in *IDPortenClient) IsUpToDate() (bool, error) {
	return isUpToDate(in)
}

func (in *IDPortenClient) MakeLabels() map[string]string {
	return labels.IDPortenLabels(in)
}

func (in *IDPortenClient) MakeDescription() string {
	return makeDescription(in)
}

func (in *IDPortenClient) ToClientRegistration() types.ClientRegistration {
	if in.Spec.AccessTokenLifetime == nil {
		lifetime := IDPortenDefaultAccessTokenLifetimeSeconds
		in.Spec.AccessTokenLifetime = &lifetime
	}
	if in.Spec.SessionLifetime == nil {
		lifetime := IDPortenDefaultSessionLifetimeSeconds
		in.Spec.SessionLifetime = &lifetime
	}
	if len(in.Spec.ClientURI) == 0 {
		in.Spec.ClientURI = IDPortenDefaultClientURI
	}
	if in.Spec.PostLogoutRedirectURIs == nil || len(in.Spec.PostLogoutRedirectURIs) == 0 {
		in.Spec.PostLogoutRedirectURIs = []string{IDPortenDefaultPostLogoutRedirectURI}
	}
	return types.ClientRegistration{
		AccessTokenLifetime:               *in.Spec.AccessTokenLifetime,
		ApplicationType:                   types.ApplicationTypeWeb,
		AuthorizationLifeTime:             *in.Spec.SessionLifetime, // should be at minimum be equal to RefreshTokenLifetime
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
		RefreshTokenLifetime: *in.Spec.SessionLifetime,
		RefreshTokenUsage:    types.RefreshTokenUsageOneTime,
		Scopes: []string{
			"openid", "profile",
		},
		TokenEndpointAuthMethod: types.TokenEndpointAuthMethodPrivateKeyJwt,
	}
}
