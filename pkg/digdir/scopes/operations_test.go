package scopes_test

import (
	"testing"

	naisiov1 "github.com/nais/liberator/pkg/apis/nais.io/v1"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/utils/ptr"

	"github.com/nais/digdirator/pkg/clients"
	"github.com/nais/digdirator/pkg/config"
	"github.com/nais/digdirator/pkg/digdir/scopes"
	"github.com/nais/digdirator/pkg/digdir/types"
)

func TestGenerateDuplicateName(t *testing.T) {
	exposed := []naisiov1.ExposedScope{
		{
			Name:    "foo.read",
			Enabled: true,
			Product: "arbeid",
			Consumers: []naisiov1.ExposedScopeConsumer{
				{
					Orgno: "0000000000",
				},
			},
			// Separator not set; defaults to "."
		},
		{
			Name:    "foo.read",
			Enabled: true,
			Product: "arbeid",
			Consumers: []naisiov1.ExposedScopeConsumer{
				{
					Orgno: "1111111111",
				},
			},
			Separator: ptr.To("/"),
		},
	}

	operations := scopes.Generate(nil, exposed)
	assert.Len(t, operations.ToCreate, 2)
	assert.Empty(t, operations.ToUpdate)
}

func TestScopeFiltering(t *testing.T) {
	scope1 := "test/existingscope"
	scope2 := "test.existingscope2"
	scope3 := "test/newscope"

	cfg := &config.Config{ClusterName: "test-cluster"}
	meta := metav1.ObjectMeta{
		Name:      "test-app",
		Namespace: "test-namespace",
	}
	client := minimalMaskinportenWithScopeInternalExternalClient(meta, createExposedScopes(scope1, scope2, scope3))
	require.Len(t, client.Spec.Scopes.ExposedScopes, 3)

	// First case:
	// with legacy scopes used on-prem
	// description: cluster:namespace:app.scope/api
	// subscope: scope/api
	scopeRegistration1 := clients.ToScopeRegistration(client, client.Spec.Scopes.ExposedScopes[0], cfg)
	assert.Equal(t, "arbeid - test-cluster:test-namespace:test-app", scopeRegistration1.Description)
	assert.Equal(t, "arbeid/test/existingscope", scopeRegistration1.Subscope)

	// Second case new format
	// description: cluster:team:app.scope
	// subscope: team:app.scope
	scopeRegistration2 := clients.ToScopeRegistration(client, client.Spec.Scopes.ExposedScopes[1], cfg)
	assert.Equal(t, scopes.Description(&meta, "test-cluster", "arbeid"), scopeRegistration2.Description)
	assert.Equal(t, "arbeid:test.existingscope2", scopeRegistration2.Subscope)

	existingRegistrations := make([]types.ScopeRegistration, 0)
	existingRegistrations = append(existingRegistrations, scopeRegistration1)
	existingRegistrations = append(existingRegistrations, scopeRegistration2)
	existingRegistrations = append(existingRegistrations, types.ScopeRegistration{
		Description: "scope is manually managed outside of digdirator",
		Name:        "nav:not/owned",
	})

	operations := scopes.Generate(existingRegistrations, client.Spec.Scopes.ExposedScopes)
	expectedConsumers := []naisiov1.ExposedScopeConsumer{{Orgno: "1010101010"}}

	require.Len(t, operations.ToCreate, 1)
	assert.Equal(t, "test/newscope", operations.ToCreate[0].Name)
	assert.Equal(t, "arbeid", operations.ToCreate[0].Product)
	assert.ElementsMatch(t, expectedConsumers, operations.ToCreate[0].Consumers)
	assert.True(t, operations.ToCreate[0].Enabled)

	require.Len(t, operations.ToUpdate, 2)
	assert.Equal(t, "arbeid/test/existingscope", operations.ToUpdate[0].ScopeRegistration.Subscope)
	assert.Equal(t, "arbeid:test.existingscope2", operations.ToUpdate[1].ScopeRegistration.Subscope)
	assert.Equal(t, "test/existingscope", operations.ToUpdate[0].CurrentScope.Name)
	assert.Equal(t, "test.existingscope2", operations.ToUpdate[1].CurrentScope.Name)
	assert.ElementsMatch(t, expectedConsumers, operations.ToUpdate[0].CurrentScope.Consumers)
	assert.ElementsMatch(t, expectedConsumers, operations.ToUpdate[1].CurrentScope.Consumers)
	assert.True(t, operations.ToUpdate[0].CurrentScope.Enabled)
	assert.True(t, operations.ToUpdate[1].CurrentScope.Enabled)
	assert.True(t, operations.ToUpdate[0].ScopeRegistration.Active)
	assert.True(t, operations.ToUpdate[1].ScopeRegistration.Active)
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
