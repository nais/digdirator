package scopes

import (
	"github.com/nais/digdirator/pkg/clients"
	"github.com/nais/digdirator/pkg/digdir/types"
	naisiov1 "github.com/nais/liberator/pkg/apis/nais.io/v1"
	"github.com/nais/liberator/pkg/kubernetes"
)

type ScopeStash struct {
	Current  []Scope
	ToCreate []naisiov1.ExposedScope
}

func (s ScopeStash) FilterScopes(actualScopesRegistrations []types.ScopeRegistration, desired clients.Instance, exposedScopes map[string]naisiov1.ExposedScope) *ScopeStash {
	for _, exposedScope := range exposedScopes {
		scopeDoNotExist := true
		for _, actual := range actualScopesRegistrations {
			if scopeExists(exposedScope.Product, exposedScope.Name, actual.Subscope) {
				scopeDoNotExist = false

				if scopeIsManaged(actual, desired, exposedScope) {
					s.Current = append(s.Current, CurrentScopeInfo(actual, exposedScope))
				}
				break
			}
		}
		if scopeDoNotExist {
			// scope do not exist, will be created
			s.ToCreate = append(s.ToCreate, exposedScope)
		}
	}
	return &s
}

func scopeExists(exposedScopeProduct, exposedScopeName, actualScopeName string) bool {
	return kubernetes.ToScope(exposedScopeProduct, exposedScopeName) == actualScopeName
}

func scopeIsManaged(actual types.ScopeRegistration, desired clients.Instance, scope naisiov1.ExposedScope) bool {
	return actual.Description == kubernetes.UniformResourceScopeName(desired, scope.Product, scope.Name)
}
