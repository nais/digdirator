package clients

import (
	"fmt"
	"reflect"

	naisiov1 "github.com/nais/liberator/pkg/apis/nais.io/v1"
	"github.com/nais/liberator/pkg/kubernetes"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"

	"github.com/nais/digdirator/pkg/config"
	"github.com/nais/digdirator/pkg/digdir/types"
	"github.com/nais/digdirator/pkg/secrets"
)

const (
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

func ToScopeRegistration(instance Instance, scope naisiov1.ExposedScope, cfg *config.Config) types.ScopeRegistration {
	switch v := instance.(type) {
	case *naisiov1.MaskinportenClient:
		return toMaskinPortenScopeRegistration(*v, scope, cfg)
	}
	return types.ScopeRegistration{}
}

func ToClientRegistration(instance Instance, cfg *config.Config) types.ClientRegistration {
	switch v := instance.(type) {
	case *naisiov1.IDPortenClient:
		return toIDPortenClientRegistration(*v, cfg)
	case *naisiov1.MaskinportenClient:
		return toMaskinPortenClientRegistration(*v, cfg)
	}
	return types.ClientRegistration{}
}

func GetIntegrationType(instance Instance) types.IntegrationType {
	switch in := instance.(type) {
	case *naisiov1.IDPortenClient:
		if in.Spec.IntegrationType != "" {
			return types.IntegrationType(in.Spec.IntegrationType)
		}
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

func GetSecretClientIDKey(instance Instance) string {
	switch instance.(type) {
	case *naisiov1.IDPortenClient:
		return secrets.IDPortenClientIDKey
	case *naisiov1.MaskinportenClient:
		return secrets.MaskinportenClientIDKey
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

func NeedsSecretRotation(instance Instance) bool {
	return instance.GetStatus().GetSynchronizationSecretName() != GetSecretName(instance)
}

func SetIDportenClientDefaultValues(in *naisiov1.IDPortenClient, cfg *config.Config) {
	var defaultValidIdportenScopes = []string{"openid", "profile"}
	var defaultValidKrrScopes = []string{"krr:global/kontaktinformasjon.read", "krr:global/digitalpost.read"}

	if in.Spec.AccessTokenLifetime == nil {
		lifetime := cfg.DigDir.Common.AccessTokenLifetime
		in.Spec.AccessTokenLifetime = &lifetime
	}
	if in.Spec.SessionLifetime == nil {
		lifetime := cfg.DigDir.Common.SessionLifetime
		in.Spec.SessionLifetime = &lifetime
	}
	if len(in.Spec.ClientURI) == 0 {
		in.Spec.ClientURI = naisiov1.IDPortenURI(cfg.DigDir.Common.ClientURI)
	}
	if in.Spec.PostLogoutRedirectURIs == nil || len(in.Spec.PostLogoutRedirectURIs) == 0 {
		in.Spec.PostLogoutRedirectURIs = []naisiov1.IDPortenURI{naisiov1.IDPortenURI(cfg.DigDir.Common.ClientURI)}
	}
	if in.Spec.IntegrationType != "" {
		switch in.Spec.IntegrationType {
		case string(types.IntegrationTypeIDPorten):
			if len(in.Spec.Scopes) == 0 {
				in.Spec.Scopes = defaultValidIdportenScopes
			}
		case string(types.IntegrationTypeKrr):
			if len(in.Spec.Scopes) == 0 {
				in.Spec.Scopes = defaultValidKrrScopes
			}
		}
	}

	if in.Spec.IntegrationType == "" {
		in.Spec.IntegrationType = string(types.IntegrationTypeIDPorten)
		in.Spec.Scopes = defaultValidIdportenScopes
	}
}

func toIDPortenClientRegistration(in naisiov1.IDPortenClient, cfg *config.Config) types.ClientRegistration {
	SetIDportenClientDefaultValues(&in, cfg)
	return types.ClientRegistration{
		AccessTokenLifetime:               *in.Spec.AccessTokenLifetime,
		ApplicationType:                   types.ApplicationTypeWeb,
		AuthorizationLifeTime:             *in.Spec.SessionLifetime, // should be at minimum be equal to RefreshTokenLifetime
		ClientName:                        cfg.DigDir.Common.ClientName,
		ClientURI:                         string(in.Spec.ClientURI),
		Description:                       kubernetes.UniformResourceName(&in.ObjectMeta, cfg.ClusterName),
		FrontchannelLogoutSessionRequired: true,
		FrontchannelLogoutURI:             string(in.Spec.FrontchannelLogoutURI),
		GrantTypes: []types.GrantType{
			types.GrantTypeAuthorizationCode,
			types.GrantTypeRefreshToken,
		},
		IntegrationType:         types.IntegrationType(in.Spec.IntegrationType),
		PostLogoutRedirectURIs:  postLogoutRedirectURIs(in.Spec.PostLogoutRedirectURIs),
		RedirectURIs:            redirectURIs(in),
		RefreshTokenLifetime:    *in.Spec.SessionLifetime,
		RefreshTokenUsage:       types.RefreshTokenUsageReuse,
		Scopes:                  in.Spec.Scopes,
		TokenEndpointAuthMethod: types.TokenEndpointAuthMethodPrivateKeyJwt,
	}
}

func toMaskinPortenClientRegistration(in naisiov1.MaskinportenClient, cfg *config.Config) types.ClientRegistration {
	return types.ClientRegistration{
		AccessTokenLifetime:               cfg.DigDir.Common.AccessTokenLifetime,
		ApplicationType:                   types.ApplicationTypeWeb,
		AuthorizationLifeTime:             cfg.DigDir.Common.SessionLifetime,
		ClientName:                        cfg.DigDir.Common.ClientName,
		ClientURI:                         cfg.DigDir.Common.ClientURI,
		Description:                       kubernetes.UniformResourceName(&in.ObjectMeta, cfg.ClusterName),
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
		Scopes:                  in.GetConsumedScopes(),
		TokenEndpointAuthMethod: types.TokenEndpointAuthMethodPrivateKeyJwt,
	}
}

func toMaskinPortenScopeRegistration(in naisiov1.MaskinportenClient, exposedScope naisiov1.ExposedScope, cfg *config.Config) types.ScopeRegistration {
	SetDefaultScopeValues(&exposedScope)
	return types.ScopeRegistration{
		Active:                     exposedScope.Enabled,
		AllowedIntegrationType:     exposedScope.AllowedIntegrations,
		AtMaxAge:                   *exposedScope.AtMaxAge,
		DelegationSource:           "",
		Name:                       "",
		AuthorizationMaxLifetime:   MaskinportenDefaultAuthorizationMaxLifetime,
		Description:                kubernetes.UniformResourceScopeName(&in.ObjectMeta, cfg.ClusterName, exposedScope.Product, exposedScope.Name),
		Prefix:                     cfg.DigDir.Maskinporten.Default.ScopePrefix,
		Subscope:                   kubernetes.ToScope(exposedScope.Product, exposedScope.Name),
		TokenType:                  types.TokenTypeSelfContained,
		Visibility:                 types.ScopeVisibilityPublic,
		RequiresPseudonymousTokens: false,
		RequiresUserAuthentication: false,
		RequiresUserConsent:        false,
	}
}

func postLogoutRedirectURIs(uris []naisiov1.IDPortenURI) []string {
	result := make([]string, 0)

	for _, uri := range uris {
		result = append(result, string(uri))
	}

	return result
}

func redirectURIs(in naisiov1.IDPortenClient) []string {
	seen := make(map[naisiov1.IDPortenURI]bool)
	res := make([]string, 0)

	for _, u := range append(in.Spec.RedirectURIs, in.Spec.RedirectURI) {
		if u != "" && !seen[u] {
			seen[u] = true
			res = append(res, string(u))
		}
	}

	return res
}

func SetDefaultScopeValues(exposedScope *naisiov1.ExposedScope) {
	if exposedScope.AllowedIntegrations == nil {
		exposedScope.AllowedIntegrations = []string{MaskinportenDefaultAllowedIntegrationType}
	}
	if exposedScope.AtMaxAge == nil {
		atAgeMax := MaskinportenDefaultAtAgeMax
		exposedScope.AtMaxAge = &atAgeMax
	}
}
