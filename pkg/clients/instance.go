package clients

import (
	"fmt"
	"github.com/nais/digdirator/pkg/annotations"
	"github.com/nais/digdirator/pkg/digdir/types"
	"github.com/nais/digdirator/pkg/secrets"
	naisiov1 "github.com/nais/liberator/pkg/apis/nais.io/v1"
	"github.com/nais/liberator/pkg/kubernetes"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"reflect"
)

const (
	// IDPortenDefaultClientURI Default values
	IDPortenDefaultClientURI                    = "https://www.nav.no"
	IDPortenDefaultPostLogoutRedirectURI        = "https://www.nav.no"
	IDPortenDefaultAccessTokenLifetimeSeconds   = 3600
	IDPortenDefaultSessionLifetimeSeconds       = 7200
	MaskinportenScopePrefix                     = "nav"
	MaskinportenDefaultAllowedIntegrationType   = "maskinporten"
	MaskinportenDefaultAtAgeMax                 = 30
	MaskinportenDefaultAuthorizationMaxLifetime = 0
)

// +kubebuilder:object:generate=false
type Instance interface {
	metav1.Object
	runtime.Object
	schema.ObjectKind
	Hash() (string, error)
	GetStatus() *naisiov1.DigdiratorStatus
	SetStatus(status naisiov1.DigdiratorStatus)
}

func ToScopeRegistration(instance Instance, scope naisiov1.ExposedScope) types.ScopeRegistration {
	switch v := instance.(type) {
	case *naisiov1.MaskinportenClient:
		return toMaskinPortenScopeRegistration(*v, scope)
	}
	return types.ScopeRegistration{}
}

func ToClientRegistration(instance Instance) types.ClientRegistration {
	switch v := instance.(type) {
	case *naisiov1.IDPortenClient:
		return toIDPortenClientRegistration(*v)
	case *naisiov1.MaskinportenClient:
		return toMaskinPortenClientRegistration(*v)
	}
	return types.ClientRegistration{}
}

func GetIntegrationType(instance Instance) types.IntegrationType {
	switch instance.(type) {
	case *naisiov1.IDPortenClient:
		return types.IntegrationTypeIDPorten
	case *naisiov1.MaskinportenClient:
		return types.IntegrationTypeMaskinporten
	}
	return types.IntegrationTypeUnknown
}

func GetInstanceType(instance Instance) string {
	return reflect.TypeOf(instance).String()
}

func GetSecretName(instance Instance) string {
	switch v := instance.(type) {
	case *naisiov1.IDPortenClient:
		return v.Spec.SecretName
	case *naisiov1.MaskinportenClient:
		return v.Spec.SecretName
	}
	return ""
}

func GetSecretJwkKey(instance Instance) string {
	switch instance.(type) {
	case *naisiov1.IDPortenClient:
		return secrets.IDPortenJwkKey
	case *naisiov1.MaskinportenClient:
		return secrets.MaskinportenJwkKey
	}
	return ""
}

func IsUpToDate(instance Instance) (bool, error) {
	newHash, err := instance.Hash()
	if err != nil {
		return false, fmt.Errorf("calculating application hash: %w", err)
	}
	return instance.GetStatus().GetSynchronizationHash() == newHash, nil
}

func ShouldUpdateSecrets(instance Instance) bool {
	return instance.GetStatus().GetSynchronizationSecretName() != GetSecretName(instance)
}

func HasSkipAnnotation(instance Instance) bool {
	switch v := instance.(type) {
	case *naisiov1.IDPortenClient:
		return annotations.HasSkipAnnotation(v)
	case *naisiov1.MaskinportenClient:
		return annotations.HasSkipAnnotation(v)
	default:
		return false
	}
}

func HasDeleteAnnotation(instance Instance) bool {
	switch v := instance.(type) {
	case *naisiov1.IDPortenClient:
		return annotations.HasDeleteAnnotation(v)
	case *naisiov1.MaskinportenClient:
		return annotations.HasDeleteAnnotation(v)
	default:
		return false
	}
}

func SetAnnotation(instance Instance, key, value string) {
	switch v := instance.(type) {
	case *naisiov1.IDPortenClient:
		annotations.Set(v, key, value)
	case *naisiov1.MaskinportenClient:
		annotations.Set(v, key, value)
	}
}

func toIDPortenClientRegistration(in naisiov1.IDPortenClient) types.ClientRegistration {
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
		Description:                       kubernetes.UniformResourceName(&in),
		FrontchannelLogoutSessionRequired: true,
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
		RefreshTokenUsage:    types.RefreshTokenUsageReuse,
		Scopes: []string{
			"openid", "profile",
		},
		TokenEndpointAuthMethod: types.TokenEndpointAuthMethodPrivateKeyJwt,
	}
}

func toMaskinPortenClientRegistration(in naisiov1.MaskinportenClient) types.ClientRegistration {
	return types.ClientRegistration{
		AccessTokenLifetime:               IDPortenDefaultAccessTokenLifetimeSeconds,
		ApplicationType:                   types.ApplicationTypeWeb,
		AuthorizationLifeTime:             IDPortenDefaultSessionLifetimeSeconds,
		ClientName:                        types.DefaultClientName,
		ClientURI:                         IDPortenDefaultClientURI,
		Description:                       kubernetes.UniformResourceName(&in),
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
		Scopes:                  in.GetUsedScopes(),
		TokenEndpointAuthMethod: types.TokenEndpointAuthMethodPrivateKeyJwt,
	}
}

func toMaskinPortenScopeRegistration(in naisiov1.MaskinportenClient, exposedScope naisiov1.ExposedScope) types.ScopeRegistration {
	if exposedScope.AllowedIntegrations == nil {
		exposedScope.AllowedIntegrations = []string{MaskinportenDefaultAllowedIntegrationType}
	}
	if exposedScope.AtAgeMax == 0 {
		exposedScope.AtAgeMax = MaskinportenDefaultAtAgeMax
	}
	uniformedScopeName := kubernetes.UniformResourceScopeName(&in, exposedScope.Name)
	return types.ScopeRegistration{
		AllowedIntegrationType:     exposedScope.AllowedIntegrations,
		AtMaxAge:                   exposedScope.AtAgeMax,
		DelegationSource:           "",
		Name:                       "",
		AuthorizationMaxLifetime:   MaskinportenDefaultAuthorizationMaxLifetime,
		Description:                uniformedScopeName,
		Prefix:                     MaskinportenScopePrefix,
		Subscope:                   kubernetes.FilterUniformedName(&in, uniformedScopeName, exposedScope.Name),
		TokenType:                  types.TokenTypeSelfContained,
		Visibility:                 types.VisibilityPublic,
		RequiresPseudonymousTokens: false,
		RequiresUserAuthentication: false,
		RequiresUserConsent:        false,
	}
}
