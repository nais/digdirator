package v1

import (
	"fmt"
	"github.com/nais/digdirator/pkg/config"
	"github.com/nais/digdirator/pkg/digdir/types"
	"github.com/nais/digdirator/pkg/labels"
	"github.com/spf13/viper"
	"gopkg.in/square/go-jose.v2"
	"reflect"
	"strings"
)

func (in *MaskinportenClient) CalculateHash() (string, error) {
	return calculateHash(in.Spec)
}

func (in *MaskinportenClient) CreateSecretData(jwk jose.JSONWebKey) (map[string]string, error) {
	wellKnownURL := viper.GetString(config.DigDirMaskinportenBaseURL) + "/.well-known/oauth-authorization-server"
	jwkJson, err := jwk.MarshalJSON()
	if err != nil {
		return nil, fmt.Errorf("marshalling JWK: %w", err)
	}
	return map[string]string{
		MaskinportenJwkKey:       string(jwkJson),
		MaskinportenWellKnownURL: wellKnownURL,
		MaskinportenClientID:     in.GetStatus().GetClientID(),
		MaskinportenScopes:       strings.Join(in.Spec.Scopes, " "),
	}, nil
}

func (in *MaskinportenClient) GetIntegrationType() types.IntegrationType {
	return types.IntegrationTypeMaskinporten
}

func (in *MaskinportenClient) GetInstanceType() string {
	return reflect.TypeOf(in).String()
}

func (in *MaskinportenClient) GetSecretMapKey() string {
	return MaskinportenJwkKey
}

func (in *MaskinportenClient) GetSecretName() string {
	return in.Spec.SecretName
}

func (in *MaskinportenClient) GetStatus() *ClientStatus {
	return &in.Status
}

func (in *MaskinportenClient) HasFinalizer(finalizerName string) bool {
	return hasFinalizer(in, finalizerName)
}

func (in *MaskinportenClient) IsBeingDeleted() bool {
	return isBeingDeleted(in)
}

func (in *MaskinportenClient) IsHashUnchanged() (bool, error) {
	return isHashUnchanged(in)
}

func (in *MaskinportenClient) MakeLabels() map[string]string {
	return labels.MaskinportenLabels(in)
}

func (in *MaskinportenClient) MakeDescription() string {
	return makeDescription(in)
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
