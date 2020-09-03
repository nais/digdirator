package idporten

import "gopkg.in/square/go-jose.v2"

type ApplicationType string

const (
	ApplicationTypeNative  ApplicationType = "native"
	ApplicationTypeWeb     ApplicationType = "web"
	ApplicationTypeBrowser ApplicationType = "browser"
)

type GrantType string

const (
	GrantTypeAuthorizationCode GrantType = "authorization_code"
	GrantTypeImplicit          GrantType = "implicit"
	GrantTypeRefreshToken      GrantType = "refresh_token"
	GrantTypeJwtBearer         GrantType = "urn:ietf:params:oauth:grant-type:jwt-bearer"
)

type IntegrationType string

const (
	IntegrationTypeApiKlient    IntegrationType = "api_klient"
	IntegrationTypeIDPorten     IntegrationType = "idporten"
	IntegrationTypeMaskinporten IntegrationType = "maskinporten"
)

type RefreshTokenUsage string

const (
	RefreshTokenUsageOneTime RefreshTokenUsage = "ONETIME"
	RefreshTokenUsageReuse   RefreshTokenUsage = "REUSE"
)

type TokenEndpointAuthMethod string

const (
	TokenEndpointAuthMethodClientSecretPost  TokenEndpointAuthMethod = "client_secret_post"
	TokenEndpointAuthMethodClientSecretBasic TokenEndpointAuthMethod = "client_secret_basic"
	TokenEndpointAuthMethodPrivateKeyJwt     TokenEndpointAuthMethod = "private_key_jwt"
	TokenEndpointAuthMethodNone              TokenEndpointAuthMethod = "none"
)

type ClientRegistration struct {
	AccessTokenLifetime               int                     `json:"access_token_lifetime"`
	ApplicationType                   ApplicationType         `json:"application_type"`
	AuthorizationLifeTime             int                     `json:"authorization_lifetime"`
	ClientID                          string                  `json:"client_id,omitempty"`
	ClientName                        string                  `json:"client_name"`
	ClientOrgno                       string                  `json:"client_orgno,omitempty"`
	ClientURI                         string                  `json:"clientURI"`
	Description                       string                  `json:"description"`
	FrontchannelLogoutSessionRequired bool                    `json:"frontchannel_logout_session_required"`
	FrontchannelLogoutURI             string                  `json:"frontchannel_logout_uri"`
	GrantTypes                        []GrantType             `json:"grant_types"`
	IntegrationType                   IntegrationType         `json:"integration_type"`
	PostLogoutRedirectURIs            []string                `json:"post_logout_redirect_uris"`
	RedirectURIs                      []string                `json:"redirect_uris"`
	RefreshTokenLifetime              int                     `json:"refresh_token_lifetime"`
	RefreshTokenUsage                 RefreshTokenUsage       `json:"refresh_token_usage"`
	Scopes                            []string                `json:"scopes"`
	TokenEndpointAuthMethod           TokenEndpointAuthMethod `json:"token_endpoint_auth_method"`
}

type RegisterJwksResponse struct {
	Created     string             `json:"created"`
	LastUpdated string             `json:"last_updated"`
	Keys        jose.JSONWebKeySet `json:"keys"`
}
