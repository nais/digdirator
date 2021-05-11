package scopes

import (
	"github.com/nais/digdirator/pkg/clients"
	"github.com/nais/digdirator/pkg/digdir/types"
	naisiov1 "github.com/nais/liberator/pkg/apis/nais.io/v1"
	"github.com/nais/liberator/pkg/kubernetes"
)

type FilteredScopeContainer struct {
	ExposedScopes []naisiov1.ExposedScope
	CurrentScopes []Scope
	ScopeToCreate []naisiov1.ExposedScope
}

func NewFilterForScope(scope []naisiov1.ExposedScope) FilteredScopeContainer {
	return FilteredScopeContainer{
		ExposedScopes: scope,
		CurrentScopes: []Scope{},
		ScopeToCreate: make([]naisiov1.ExposedScope, 0),
	}
}

func (s FilteredScopeContainer) FilterScopes(actualScopesRegistrations []types.ScopeRegistration, desired clients.Instance) *FilteredScopeContainer {
	currentSubscopes := make([]string, 0)

	for _, ExposedScope := range s.ExposedScopes {
		scopeDoNotExist := true
		for _, actual := range actualScopesRegistrations {
			if scopeExistsInList(ExposedScope.Name, actual.Name) {
				scopeDoNotExist = false

				// cluster:namespace:appnavn/scope.read
				if scopeMatches(actual, desired) {
					// match is found for already registered scope for team:namespace
					currentSubscopes = append(currentSubscopes, actual.Name)
					s.CurrentScopes = append(s.CurrentScopes, CreateScope(ExposedScope.Consumers, actual))
				}
			}
		}
		if scopeDoNotExist {
			// scope do not exist, will be created
			currentSubscopes = append(currentSubscopes, ExposedScope.Name)
			s.ScopeToCreate = append(s.ScopeToCreate, ExposedScope)
		}
	}
	// Setting current scopes for an app
	desired.GetStatus().SetApplicationScopes(currentSubscopes)
	return &s
}

func scopeExistsInList(ExposedScopeName string, actualScopeName string) bool {
	return ExposedScopeName == actualScopeName
}

func scopeMatches(actual types.ScopeRegistration, desired clients.Instance) bool {
	return actual.Description == kubernetes.UniformResourceScopeName(desired, actual.Name)
}
