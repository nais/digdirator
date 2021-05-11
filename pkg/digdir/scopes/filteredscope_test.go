package scopes

import (
	"github.com/nais/digdirator/pkg/clients"
	"github.com/nais/digdirator/pkg/digdir/types"
	naisiov1 "github.com/nais/liberator/pkg/apis/nais.io/v1"
	"github.com/nais/liberator/pkg/kubernetes"
	"github.com/stretchr/testify/assert"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"testing"
)

func TestScopeFiltering(t *testing.T) {
	currentScope := "test/scope"
	currentScope2 := "test.scope2"
	noneExistingScope := "scope/nr2"

	currentObjectMeta := metav1.ObjectMeta{
		Name:        "test-app",
		Namespace:   "test-namespace",
		ClusterName: "test-cluster",
	}

	currentExternals := []naisiov1.ExposedScope{
		{
			Name: currentScope,
			Consumers: []naisiov1.ExposedScopeConsumer{
				{
					Orgno: "1010101010",
				},
			},
		},
		{
			Name: currentScope2,
			Consumers: []naisiov1.ExposedScopeConsumer{
				{
					Orgno: "1010101010",
				},
			},
		},
		{
			Name: noneExistingScope,
			Consumers: []naisiov1.ExposedScopeConsumer{
				{
					Orgno: "1010101010",
				},
			},
		},
	}

	currentMaskinportenClient := minimalMaskinportenWithScopeInternalExternalClient(currentObjectMeta, currentExternals)

	scopeContainer := NewFilterForScope(currentMaskinportenClient.GetExposedScopes())

	actualScopeRegistrations := make([]types.ScopeRegistration, 0)

	// First case:
	// with legacy scopes used on-prem
	// description: cluster:namespace:app.scope/api
	// subscope: scope/api
	scoperegistration1 := scopeRegistration(*currentMaskinportenClient, currentMaskinportenClient.GetExposedScopes()[0], currentScope)
	assert.Equal(t, kubernetes.UniformResourceScopeName(&currentObjectMeta, scoperegistration1.Name), scoperegistration1.Description)
	assert.Equal(t, currentScope, scoperegistration1.Subscope)

	// Secound case new format
	// description: cluster:team:app.scope
	// subscope: team:app.scope
	scoperegistration2 := scopeRegistration(*currentMaskinportenClient, currentMaskinportenClient.GetExposedScopes()[1], currentScope2)
	assert.Equal(t, kubernetes.UniformResourceScopeName(&currentObjectMeta, scoperegistration2.Name), scoperegistration2.Description)
	assert.Equal(t, kubernetes.UniformResourceScopeName(&currentObjectMeta, scoperegistration2.Name), scoperegistration2.Subscope)

	// scopes owned by current application
	actualScopeRegistrations = append(actualScopeRegistrations, scoperegistration1)
	actualScopeRegistrations = append(actualScopeRegistrations, scoperegistration2)

	// FilteredScopeContainer not managed by digdirator should be ignored
	actualScopeRegistrations = append(actualScopeRegistrations, types.ScopeRegistration{
		Description: "some: random description:",
		Name:        "nav:not/owned",
	})

	result := *scopeContainer.FilterScopes(actualScopeRegistrations, currentMaskinportenClient)

	// Scopes not existing in digidir but wil be added to managed
	assert.Equal(t, 1, len(result.ScopeToCreate))
	scopesToCreate := result.ScopeToCreate[0]
	assert.Equal(t, noneExistingScope, scopesToCreate.Name)
	assert.Equal(t, 1, len(scopesToCreate.Consumers))

	// Scopes existing, owned and used by current namespace
	assert.Equal(t, 2, len(result.CurrentScopes))
	currentScopes1 := result.CurrentScopes[0]
	assert.Equal(t, currentScope, currentScopes1.ScopeRegistration.Name)
	currentScopes2 := result.CurrentScopes[1]
	assert.Equal(t, currentScope2, currentScopes2.ScopeRegistration.Name)
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

func scopeRegistration(in naisiov1.MaskinportenClient, exposedScope naisiov1.ExposedScope, scope string) types.ScopeRegistration {
	uniformedName := kubernetes.UniformResourceScopeName(&in, exposedScope.Name)
	return types.ScopeRegistration{
		AllowedIntegrationType:     exposedScope.AllowedIntegrations,
		AtMaxAge:                   exposedScope.AtAgeMax,
		DelegationSource:           "",
		Name:                       scope,
		AuthorizationMaxLifetime:   clients.MaskinportenDefaultAuthorizationMaxLifetime,
		Description:                uniformedName,
		Prefix:                     clients.MaskinportenScopePrefix,
		Subscope:                   kubernetes.FilterUniformedName(&in, uniformedName, exposedScope.Name),
		TokenType:                  types.TokenTypeSelfContained,
		Visibility:                 types.VisibilityPublic,
		RequiresPseudonymousTokens: false,
		RequiresUserAuthentication: false,
		RequiresUserConsent:        false,
	}
}
