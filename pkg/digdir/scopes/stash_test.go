package scopes

import (
	"fmt"
	"testing"

	naisiov1 "github.com/nais/liberator/pkg/apis/nais.io/v1"
	"github.com/nais/liberator/pkg/kubernetes"
	"github.com/stretchr/testify/assert"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/nais/digdirator/pkg/clients"
	"github.com/nais/digdirator/pkg/digdir/types"
)

func TestScopeFilteringWithNewScopeAndOneExistingOne(t *testing.T) {
	currentScope := "test/scope"
	currentObjectMeta := metaObject()
	exposedScopes := createExposedScopes(currentScope)
	currentMaskinportenClient := minimalMaskinportenWithScopeInternalExternalClient(currentObjectMeta, exposedScopes)
	scopeContainer := ScopeStash{}

	scopeRegistration := clients.ToScopeRegistration(currentMaskinportenClient, currentMaskinportenClient.GetExposedScopes()[currentScope])
	scopeRegistration.Name = fmt.Sprintf("nav:%s", currentScope)
	scopeRegistration.Active = true
	assert.Equal(t, kubernetes.UniformResourceScopeName(&currentObjectMeta, currentObjectMeta.ClusterName, "arbeid", currentScope), scopeRegistration.Description)
	assert.Equal(t, kubernetes.ToScope("arbeid", currentScope), scopeRegistration.Subscope)
	assert.True(t, scopeRegistration.Active)

	actualScopeRegistrations := make([]types.ScopeRegistration, 0)
	actualScopeRegistrations = append(actualScopeRegistrations, scopeRegistration)

	result := *scopeContainer.FilterScopes(actualScopeRegistrations, currentMaskinportenClient, currentMaskinportenClient.GetExposedScopes())

	assert.Equal(t, 0, len(result.ToCreate))
	assert.Equal(t, 1, len(result.ToUpdate))
	assert.Equal(t, currentScope, result.ToUpdate[0].CurrentScope.Name)
	assert.True(t, result.ToUpdate[0].CurrentScope.Enabled)
}

func TestScopeFiltering(t *testing.T) {
	currentScope := "test/scope"
	currentScope2 := "test.scope2"
	noneExistingScope := "scope/nr2"
	currentObjectMeta := metaObject()
	currentExternals := createExposedScopes(currentScope, currentScope2, noneExistingScope)
	currentMaskinportenClient := minimalMaskinportenWithScopeInternalExternalClient(currentObjectMeta, currentExternals)
	scopeContainer := ScopeStash{}
	actualScopeRegistrations := make([]types.ScopeRegistration, 0)

	// First case:
	// with legacy scopes used on-prem
	// description: cluster:namespace:app.scope/api
	// subscope: scope/api
	scopeRegistration1 := clients.ToScopeRegistration(currentMaskinportenClient, currentMaskinportenClient.GetExposedScopes()[currentScope])
	scopeRegistration1.Name = fmt.Sprintf("nav:%s", currentScope)
	assert.Equal(t, kubernetes.UniformResourceScopeName(&currentObjectMeta, currentObjectMeta.GetClusterName(), "arbeid", currentScope), scopeRegistration1.Description)
	assert.Equal(t, kubernetes.ToScope("arbeid", currentScope), scopeRegistration1.Subscope)

	// Secound case new format
	// description: cluster:team:app.scope
	// subscope: team:app.scope
	scopeRegistration2 := clients.ToScopeRegistration(currentMaskinportenClient, currentMaskinportenClient.GetExposedScopes()[currentScope2])
	scopeRegistration2.Name = fmt.Sprintf("nav:%s", currentScope2)
	assert.Equal(t, kubernetes.UniformResourceScopeName(&currentObjectMeta, currentObjectMeta.GetClusterName(), "arbeid", currentScope2), scopeRegistration2.Description)
	assert.Equal(t, "arbeid:test.scope2", scopeRegistration2.Subscope)

	// add scopes owned by current application
	actualScopeRegistrations = append(actualScopeRegistrations, scopeRegistration1)
	actualScopeRegistrations = append(actualScopeRegistrations, scopeRegistration2)

	// ScopeStash not managed by digdirator should be ignored
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
	assert.Equal(t, 2, len(result.ToUpdate))

	validRegistrations := map[string]string{
		currentScope:  currentScope,
		currentScope2: currentScope2,
	}

	for _, v := range result.ToUpdate {
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
				ConsumedScopes: []naisiov1.ConsumedScope{
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
			Product: "arbeid",
			Consumers: []naisiov1.ExposedScopeConsumer{
				{
					Orgno: "1010101010",
				},
			},
		})
	}
	return exposed
}
