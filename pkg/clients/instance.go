package clients

import (
	"reflect"
	"time"

	naisiov1 "github.com/nais/liberator/pkg/apis/nais.io/v1"
	"github.com/nais/liberator/pkg/kubernetes"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/utils/ptr"

	"github.com/nais/digdirator/pkg/config"
	"github.com/nais/digdirator/pkg/digdir/scopes"
	"github.com/nais/digdirator/pkg/digdir/types"
	"github.com/nais/digdirator/pkg/secrets"
)

const (
	AnnotationResynchronize = "digdir.nais.io/resync"
	AnnotationRotate        = "digdir.nais.io/rotate"

	MaskinportenDefaultAllowedIntegrationType   = "maskinporten"
	MaskinportenDefaultAtAgeMax                 = 30
	MaskinportenDefaultAuthorizationMaxLifetime = 0

	StaleSyncThresholdDuration = 7 * 24 * time.Hour
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

// TODO: use this as an alternative fallback for getting a client ID?
func GetSecretClientIDKey(instance Instance) string {
	switch instance.(type) {
	case *naisiov1.IDPortenClient:
		return secrets.IDPortenClientIDKey
	case *naisiov1.MaskinportenClient:
		return secrets.MaskinportenClientIDKey
	}
	return ""
}

func IsUpToDate(instance Instance) bool {
	status := instance.GetStatus()
	if status == nil {
		return false
	}

	observedGeneration := ptr.Deref(status.ObservedGeneration, 0)
	if observedGeneration == 0 {
		return false
	}
	generationChanged := instance.GetGeneration() != observedGeneration

	a := instance.GetAnnotations()
	resync := hasAnnotation(a, AnnotationResynchronize)
	rotate := hasAnnotation(a, AnnotationRotate)

	return !generationChanged && !resync && !rotate && !isStale(status)
}

func NeedsSecretRotation(instance Instance) bool {
	nameChanged := instance.GetStatus().SynchronizationSecretName != GetSecretName(instance)
	hasRotateAnnotation := hasAnnotation(instance.GetAnnotations(), AnnotationRotate)

	return nameChanged || hasRotateAnnotation
}

func GetIDPortenDefaultScopes(integrationType string) []string {
	switch integrationType {
	case string(types.IntegrationTypeIDPorten), string(types.IntegrationTypeApiKlient):
		return []string{"openid", "profile"}
	case string(types.IntegrationTypeKrr):
		return []string{"krr:global/kontaktinformasjon.read", "krr:global/digitalpost.read"}
	}
	return []string{}
}

func SetIDportenClientDefaultValues(in *naisiov1.IDPortenClient, cfg *config.Config) {
	if in.Spec.ApplicationType == "" {
		in.Spec.ApplicationType = string(types.ApplicationTypeWeb)
	}
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
	if len(in.Spec.PostLogoutRedirectURIs) == 0 {
		in.Spec.PostLogoutRedirectURIs = []naisiov1.IDPortenURI{naisiov1.IDPortenURI(cfg.DigDir.Common.ClientURI)}
	}

	if in.Spec.IntegrationType == "" {
		in.Spec.IntegrationType = string(types.IntegrationTypeIDPorten)
	}

	if len(in.Spec.Scopes) == 0 {
		in.Spec.Scopes = GetIDPortenDefaultScopes(in.Spec.IntegrationType)
	}
}

func toIDPortenClientRegistration(in naisiov1.IDPortenClient, cfg *config.Config) types.ClientRegistration {
	SetIDportenClientDefaultValues(&in, cfg)

	integrationType := types.IntegrationType(in.Spec.IntegrationType)
	if in.Spec.IntegrationType == string(types.IntegrationTypeMaskinporten) {
		integrationType = types.IntegrationTypeIDPorten
	}

	clientName := in.Spec.ClientName
	if clientName == "" {
		clientName = cfg.DigDir.Common.ClientName
	}

	ssoDisabled := false
	if in.Spec.SSODisabled != nil && *in.Spec.SSODisabled {
		ssoDisabled = true
	}

	// Set token endpoint auth method based on application type
	tokenEndpointAuthMethod := types.TokenEndpointAuthMethodPrivateKeyJwt
	if in.Spec.ApplicationType != string(types.ApplicationTypeWeb) {
		tokenEndpointAuthMethod = types.TokenEndpointAuthMethodNone
	}

	return types.ClientRegistration{
		AccessTokenLifetime:               *in.Spec.AccessTokenLifetime,
		ApplicationType:                   types.ApplicationType(in.Spec.ApplicationType),
		AuthorizationLifeTime:             *in.Spec.SessionLifetime, // should be at minimum be equal to RefreshTokenLifetime
		ClientName:                        clientName,
		ClientURI:                         string(in.Spec.ClientURI),
		Description:                       kubernetes.UniformResourceName(&in.ObjectMeta, cfg.ClusterName),
		FrontchannelLogoutSessionRequired: true,
		FrontchannelLogoutURI:             string(in.Spec.FrontchannelLogoutURI),
		GrantTypes: []types.GrantType{
			types.GrantTypeAuthorizationCode,
			types.GrantTypeRefreshToken,
		},
		IntegrationType:         integrationType,
		PostLogoutRedirectURIs:  postLogoutRedirectURIs(in.Spec.PostLogoutRedirectURIs),
		RedirectURIs:            redirectURIs(in),
		RefreshTokenLifetime:    *in.Spec.SessionLifetime,
		RefreshTokenUsage:       types.RefreshTokenUsageOneTime,
		Scopes:                  in.Spec.Scopes,
		SSODisabled:             ssoDisabled,
		TokenEndpointAuthMethod: tokenEndpointAuthMethod,
	}
}

func toMaskinPortenClientRegistration(in naisiov1.MaskinportenClient, cfg *config.Config) types.ClientRegistration {
	clientName := in.Spec.ClientName
	if clientName == "" {
		clientName = cfg.DigDir.Common.ClientName
	}

	return types.ClientRegistration{
		AccessTokenLifetime:               cfg.DigDir.Common.AccessTokenLifetime,
		ApplicationType:                   types.ApplicationTypeWeb,
		AuthorizationLifeTime:             cfg.DigDir.Common.SessionLifetime,
		ClientName:                        clientName,
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
		RefreshTokenUsage:       types.RefreshTokenUsageOneTime,
		Scopes:                  in.GetConsumedScopes(),
		TokenEndpointAuthMethod: types.TokenEndpointAuthMethodPrivateKeyJwt,
	}
}

func toMaskinPortenScopeRegistration(in naisiov1.MaskinportenClient, exposedScope naisiov1.ExposedScope, cfg *config.Config) types.ScopeRegistration {
	allowedIntegrations := []string{MaskinportenDefaultAllowedIntegrationType}
	if len(exposedScope.AllowedIntegrations) > 0 {
		allowedIntegrations = exposedScope.AllowedIntegrations
	}

	accessTokenMaxAge := MaskinportenDefaultAtAgeMax
	if exposedScope.AtMaxAge != nil {
		accessTokenMaxAge = *exposedScope.AtMaxAge
	}

	delegationSource := ""
	if exposedScope.DelegationSource != nil {
		name := *exposedScope.DelegationSource
		source, ok := cfg.DigDir.Maskinporten.DelegationSources[name]
		if ok {
			delegationSource = source.Issuer
		}
	}

	accessibleForAll := false
	if exposedScope.AccessibleForAll != nil && *exposedScope.AccessibleForAll {
		accessibleForAll = true
	}

	visibility := types.ScopeVisibilityPublic
	if exposedScope.Visibility != nil && *exposedScope.Visibility == "private" {
		visibility = types.ScopeVisibilityPrivate
	}

	return types.ScopeRegistration{
		AccessibleForAll:           accessibleForAll,
		Active:                     exposedScope.Enabled,
		AllowedIntegrationType:     allowedIntegrations,
		AtMaxAge:                   accessTokenMaxAge,
		DelegationSource:           delegationSource,
		Name:                       "",
		AuthorizationMaxLifetime:   MaskinportenDefaultAuthorizationMaxLifetime,
		Description:                scopes.Description(&in.ObjectMeta, cfg.ClusterName, exposedScope.Product),
		Prefix:                     cfg.DigDir.Maskinporten.Default.ScopePrefix,
		Subscope:                   scopes.Subscope(exposedScope),
		TokenType:                  types.TokenTypeSelfContained,
		Visibility:                 visibility,
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

func hasAnnotation(annotations map[string]string, want string) bool {
	value, found := annotations[want]
	return found && value == "true"
}

func isStale(status *naisiov1.DigdiratorStatus) bool {
	lastSync := status.SynchronizationTime
	if lastSync == nil {
		return false
	}

	return time.Since(lastSync.Time) > StaleSyncThresholdDuration
}
