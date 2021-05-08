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
	currentScope := "nav:test/scope"
	currentScope2 := "nav:test/scope2"
	noneExistingScope := "nav:test/scope/nr2"
	ownedScope := "nav:test/scope/nr3"

	currentObjectMeta := metav1.ObjectMeta{
		Name:        "test-app",
		Namespace:   "test-namespace",
		ClusterName: "test-cluster",
	}

	currentExternals := []naisiov1.ExternalScope{
		{
			Name: currentScope,
			Consumers: []naisiov1.ExternalScopeConsumer{
				{
					Orgno: "1010101010",
				},
			},
		},
		{
			Name: currentScope2,
			Consumers: []naisiov1.ExternalScopeConsumer{
				{
					Orgno: "1010101010",
				},
			},
		},
		{
			Name: noneExistingScope,
			Consumers: []naisiov1.ExternalScopeConsumer{
				{
					Orgno: "1010101010",
				},
			},
		},
		{
			Name: ownedScope,
			Consumers: []naisiov1.ExternalScopeConsumer{
				{
					Orgno: "1010101010",
				},
			},
		},
	}

	currentMaskinportenClient := minimalMaskinportenWithScopeInternalExternalClient(currentObjectMeta, currentExternals)

	ownedObjectMeta := metav1.ObjectMeta{
		Name:        "owned-app",
		Namespace:   "owned-namespace",
		ClusterName: "test-cluster",
	}
	ownedExternals := []naisiov1.ExternalScope{
		{
			Name: ownedScope,
			Consumers: []naisiov1.ExternalScopeConsumer{
				{
					Orgno: "11111111",
				},
			},
		},
	}
	ownedMaskinportenClient := minimalMaskinportenWithScopeInternalExternalClient(ownedObjectMeta, ownedExternals)

	scopeContainer := NewFilterForScope(currentMaskinportenClient.GetExternalScopes())

	actualScopeRegistrations := make([]types.ScopeRegistration, 0)
	// scopes owned by current namespace
	actualScopeRegistrations = append(actualScopeRegistrations, scopeRegistration(*currentMaskinportenClient, currentMaskinportenClient.GetExternalScopes()[0], currentScope))
	actualScopeRegistrations = append(actualScopeRegistrations, scopeRegistration(*currentMaskinportenClient, currentMaskinportenClient.GetExternalScopes()[1], currentScope2))
	// FilteredScopeContainer owned by another namespace
	actualScopeRegistrations = append(actualScopeRegistrations, scopeRegistration(*ownedMaskinportenClient, ownedMaskinportenClient.GetExternalScopes()[0], ownedScope))
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

	// Scopes existing but used by another namespace
	assert.Equal(t, 1, len(result.OwnedScopes))
	ownedScopes := result.OwnedScopes[0]
	assert.Equal(t, ownedScope, ownedScopes.Name)

	// Scopes existing, owned and used by current namespace
	assert.Equal(t, 2, len(result.CurrentScopes))
	currentScopes1 := result.CurrentScopes[0]
	assert.Equal(t, currentScope, currentScopes1.ScopeRegistration.Name)
	currentScopes2 := result.CurrentScopes[1]
	assert.Equal(t, currentScope2, currentScopes2.ScopeRegistration.Name)
}

func minimalMaskinportenWithScopeInternalExternalClient(meta metav1.ObjectMeta, scope []naisiov1.ExternalScope) *naisiov1.MaskinportenClient {
	return &naisiov1.MaskinportenClient{
		ObjectMeta: meta,
		Spec: naisiov1.MaskinportenClientSpec{
			Scope: naisiov1.MaskinportenScopeSpec{
				Internal: []naisiov1.InternalScope{
					{
						Name: "some-scope",
					},
				},
				External: scope,
			},
		},
	}
}

func scopeRegistration(in naisiov1.MaskinportenClient, externalScope naisiov1.ExternalScope, scope string) types.ScopeRegistration {
	return types.ScopeRegistration{
		AllowedIntegrationType:     externalScope.AllowedIntegrations,
		AtMaxAge:                   externalScope.AtAgeMax,
		DelegationSource:           "",
		Name:                       scope,
		AuthorizationMaxLifetime:   0,
		Description:                kubernetes.UniformResourceScopeName(&in, externalScope.Name),
		Prefix:                     "nav",
		Subscope:                   clients.FilterScopePrefix(externalScope.Name),
		TokenType:                  types.TokenTypeSelfContained,
		Visibility:                 types.VisibilityPublic,
		RequiresPseudonymousTokens: false,
		RequiresUserAuthentication: false,
		RequiresUserConsent:        false,
	}
}
