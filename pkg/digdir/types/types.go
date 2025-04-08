package types

import (
	"time"

	"github.com/nais/digdirator/pkg/crypto"
)

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
	IntegrationTypeKrr          IntegrationType = "krr"
	IntegrationTypeUnknown      IntegrationType = "unknown"
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
	ClientURI                         string                  `json:"client_uri,omitempty"`
	Description                       string                  `json:"description"`
	FrontchannelLogoutSessionRequired bool                    `json:"frontchannel_logout_session_required"`
	FrontchannelLogoutURI             string                  `json:"frontchannel_logout_uri,omitempty"`
	GrantTypes                        []GrantType             `json:"grant_types"`
	IntegrationType                   IntegrationType         `json:"integration_type"`
	PostLogoutRedirectURIs            []string                `json:"post_logout_redirect_uris"`
	RedirectURIs                      []string                `json:"redirect_uris"`
	RefreshTokenLifetime              int                     `json:"refresh_token_lifetime,omitzero"`
	RefreshTokenUsage                 RefreshTokenUsage       `json:"refresh_token_usage"`
	Scopes                            []string                `json:"scopes"`
	SSODisabled                       bool                    `json:"sso_disabled"`
	TokenEndpointAuthMethod           TokenEndpointAuthMethod `json:"token_endpoint_auth_method"`
}

type JwksResponse struct {
	Created     string `json:"created"`
	LastUpdated string `json:"last_updated"`
	crypto.DigdirJwkSet
}

type ScopeAccessState string

const (
	ScopeAccessRequested ScopeAccessState = "REQUESTED"
	ScopeAccessApproved  ScopeAccessState = "APPROVED"
	ScopeAccessDenied    ScopeAccessState = "DENIED"
)

type Scope struct {
	ConsumerOrgNo string           `json:"consumer_orgno"`
	OwnerOrgNo    string           `json:"owner_orgno"`
	Scope         string           `json:"scope"`
	State         ScopeAccessState `json:"state"`
}

func (s Scope) IsAccessible() bool {
	return s.State == ScopeAccessApproved
}

type Visibility string

const (
	ScopeVisibilityInternal Visibility = "INTERNAL"
	ScopeVisibilityPrivate  Visibility = "PRIVATE"
	ScopeVisibilityPublic   Visibility = "PUBLIC"
)

type TokenType string

const (
	TokenTypeSelfContained TokenType = "SELF_CONTAINED"
	TokenTypeOpaque        TokenType = "OPAQUE"
)

type ScopeRegistration struct {
	AccessibleForAll           bool       `json:"accessible_for_all"`
	Active                     bool       `json:"active"`
	AllowedIntegrationType     []string   `json:"allowed_integration_types"`
	AtMaxAge                   int        `json:"at_max_age"`
	AuthorizationMaxLifetime   int        `json:"authorization_max_lifetime,omitempty"`
	DelegationSource           string     `json:"delegation_source,omitempty"`
	Description                string     `json:"description"`
	LongDescription            string     `json:"long_description,omitempty"`
	Name                       string     `json:"name"`
	OwnerOrgno                 string     `json:"owner_orgno,omitempty"`
	Prefix                     string     `json:"prefix"`
	RequiresPseudonymousTokens bool       `json:"requires_pseudonymous_tokens"`
	RequiresUserAuthentication bool       `json:"requires_user_authentication"`
	RequiresUserConsent        bool       `json:"requires_user_consent"`
	Subscope                   string     `json:"subscope"`
	TokenType                  TokenType  `json:"token_type"`
	Visibility                 Visibility `json:"visibility"`
}

type State string

const (
	ScopeStateRequested State = "REQUESTED"
	ScopeStateApproved  State = "APPROVED"
	ScopeStateDenied    State = "DENIED"
)

type ConsumerRegistration struct {
	ConsumerOrgno string    `json:"consumer_orgno"`
	Created       time.Time `json:"created"`
	LastUpdated   time.Time `json:"last_updated"`
	OwnerOrgno    string    `json:"owner_orgno"`
	Scope         string    `json:"scope"`
	State         State     `json:"state"`
}

type DelegationSource struct {
	Name   string `json:"name"`
	Issuer string `json:"issuer"`
}
