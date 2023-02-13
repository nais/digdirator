package scopes

import (
	naisiov1 "github.com/nais/liberator/pkg/apis/nais.io/v1"
	"github.com/nais/liberator/pkg/kubernetes"

	"github.com/nais/digdirator/pkg/clients"
	"github.com/nais/digdirator/pkg/digdir/types"
)

type ScopeStash struct {
	ToCreate []naisiov1.ExposedScope
	ToUpdate []Scope
}

func (s ScopeStash) FilterScopes(actualScopes []types.ScopeRegistration, client clients.Instance, desiredScopes map[string]naisiov1.ExposedScope, clusterName string) *ScopeStash {
	for _, desired := range desiredScopes {
		exists := false

		for _, actual := range actualScopes {
			if !scopeExists(desired.Product, desired.Name, actual.Subscope) {
				continue
			}

			exists = true
			if scopeIsManaged(client, actual, desired, clusterName) {
				s.ToUpdate = append(s.ToUpdate, CurrentScopeInfo(actual, desired))
			}

			break
		}

		if !exists {
			s.ToCreate = append(s.ToCreate, desired)
		}
	}

	return &s
}

func scopeExists(exposedScopeProduct, exposedScopeName, actualScopeName string) bool {
	return kubernetes.ToScope(exposedScopeProduct, exposedScopeName) == actualScopeName
}

func scopeIsManaged(client clients.Instance, actual types.ScopeRegistration, scope naisiov1.ExposedScope, clusterName string) bool {
	return actual.Description == kubernetes.UniformResourceScopeName(client, clusterName, scope.Product, scope.Name)
}
