package scopes

import (
	"fmt"
	"github.com/nais/digdirator/pkg/clients"
	"github.com/nais/digdirator/pkg/digdir/types"
	naisiov1 "github.com/nais/liberator/pkg/apis/nais.io/v1"
	"github.com/nais/liberator/pkg/kubernetes"
	"github.com/stretchr/testify/assert"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"testing"
)

func TestScopeFilteringWithNewScopeAndOneExistingOne(t *testing.T) {
	currentScope := "test/scope"
	currentObjectMeta := metaObject()
	exposedScopes := createExposedScopes(currentScope)
	currentMaskinportenClient := minimalMaskinportenWithScopeInternalExternalClient(currentObjectMeta, exposedScopes)
	scopeContainer := FilteredScopeContainer{}

	scopeRegistration := toScopeRegistration(*currentMaskinportenClient, currentMaskinportenClient.GetExposedScopes()[currentScope], currentScope)
	assert.Equal(t, kubernetes.UniformResourceScopeName(&currentObjectMeta, currentScope), scopeRegistration.Description)
	assert.Equal(t, currentScope, scopeRegistration.Subscope)
	assert.True(t, scopeRegistration.Active)

	actualScopeRegistrations := make([]types.ScopeRegistration, 0)
	actualScopeRegistrations = append(actualScopeRegistrations, scopeRegistration)

	result := *scopeContainer.FilterScopes(actualScopeRegistrations, currentMaskinportenClient, currentMaskinportenClient.GetExposedScopes())

	assert.Equal(t, 0, len(result.ToCreate))
	assert.Equal(t, 1, len(result.Current))
	assert.Equal(t, currentScope, result.Current[0].CurrentScope.Name)
	assert.True(t, result.Current[0].CurrentScope.Enabled)
}

func TestScopeFiltering(t *testing.T) {
	currentScope := "test/scope"
	currentScope2 := "test.scope2"
	noneExistingScope := "scope/nr2"
	currentObjectMeta := metaObject()
	currentExternals := createExposedScopes(currentScope, currentScope2, noneExistingScope)
	currentMaskinportenClient := minimalMaskinportenWithScopeInternalExternalClient(currentObjectMeta, currentExternals)
	scopeContainer := FilteredScopeContainer{}
	actualScopeRegistrations := make([]types.ScopeRegistration, 0)

	// First case:
	// with legacy scopes used on-prem
	// description: cluster:namespace:app.scope/api
	// subscope: scope/api
	scoperegistration1 := toScopeRegistration(*currentMaskinportenClient, currentMaskinportenClient.GetExposedScopes()[currentScope], currentScope)
	assert.Equal(t, kubernetes.UniformResourceScopeName(&currentObjectMeta, currentScope), scoperegistration1.Description)
	assert.Equal(t, currentScope, scoperegistration1.Subscope)

	// Secound case new format
	// description: cluster:team:app.scope
	// subscope: team:app.scope
	scoperegistration2 := toScopeRegistration(*currentMaskinportenClient, currentMaskinportenClient.GetExposedScopes()[currentScope2], currentScope2)
	assert.Equal(t, kubernetes.UniformResourceScopeName(&currentObjectMeta, currentScope2), scoperegistration2.Description)
	assert.Equal(t, "testnamespace:testapp.test.scope2", scoperegistration2.Subscope)

	// add scopes owned by current application
	actualScopeRegistrations = append(actualScopeRegistrations, scoperegistration1)
	actualScopeRegistrations = append(actualScopeRegistrations, scoperegistration2)

	// FilteredScopeContainer not managed by digdirator should be ignored
	actualScopeRegistrations = append(actualScopeRegistrations, types.ScopeRegistration{
		Description: "some: random description:",
		Name:        "nav:not/owned",
	})

	result := *scopeContainer.FilterScopes(actualScopeRegistrations, currentMaskinportenClient, currentMaskinportenClient.GetExposedScopes())

	// Scopes not existing in digidir but will be added to managed
	scopesToCreate := result.ToCreate[0]
	assert.Equal(t, 1, len(result.ToCreate))
	assert.Equal(t, noneExistingScope, scopesToCreate.Name)
	assert.Equal(t, 1, len(scopesToCreate.Consumers))

	// Scopes existing, owned and used by current application
	assert.Equal(t, 2, len(result.Current))

	validRegistrations := map[string]string{
		currentScope:  currentScope,
		currentScope2: currentScope2,
	}

	for _, v := range result.Current {
		if _, ok := validRegistrations[v.ScopeRegistration.Name]; ok {
			assert.True(t, ok, "Map should be a valid list of current scopes")
		}
	}
}

func minimalMaskinportenWithScopeInternalExternalClient(meta metav1.ObjectMeta, scope []naisiov1.ExposedScope) *naisiov1.MaskinportenClient {
	return &naisiov1.MaskinportenClient{
		ObjectMeta: meta,
		Spec: naisiov1.MaskinportenClientSpec{
			Scopes: naisiov1.MaskinportenScope{
				UsedScope: []naisiov1.UsedScope{
					{
						Name: "some.scope",
					},
				},
				ExposedScopes: scope,
			},
		},
	}
}

func metaObject() metav1.ObjectMeta {
	return metav1.ObjectMeta{
		Name:        "test-app",
		Namespace:   "test-namespace",
		ClusterName: "test-cluster",
	}
}

func createExposedScopes(scopeNames ...string) []naisiov1.ExposedScope {
	exposed := make([]naisiov1.ExposedScope, 0)
	for _, s := range scopeNames {
		exposed = append(exposed, naisiov1.ExposedScope{
			Name:    s,
			Enabled: true,
			Consumers: []naisiov1.ExposedScopeConsumer{
				{
					Orgno: "1010101010",
				},
			},
		})
	}
	return exposed
}

func toScopeRegistration(in naisiov1.MaskinportenClient, exposedScope naisiov1.ExposedScope, scope string) types.ScopeRegistration {
	return types.ScopeRegistration{
		AllowedIntegrationType:     exposedScope.AllowedIntegrations,
		AtMaxAge:                   exposedScope.AtAgeMax,
		Active:                     true,
		DelegationSource:           "",
		Name:                       fmt.Sprintf("nav:%s", scope),
		AuthorizationMaxLifetime:   clients.MaskinportenDefaultAuthorizationMaxLifetime,
		Description:                kubernetes.UniformResourceScopeName(&in, exposedScope.Name),
		Prefix:                     clients.MaskinportenScopePrefix,
		Subscope:                   kubernetes.FilterUniformedName(&in, exposedScope.Name),
		TokenType:                  types.TokenTypeSelfContained,
		Visibility:                 types.VisibilityPublic,
		RequiresPseudonymousTokens: false,
		RequiresUserAuthentication: false,
		RequiresUserConsent:        false,
	}
}
