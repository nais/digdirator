package scopes

import (
	"github.com/nais/digdirator/pkg/clients"
	"github.com/nais/digdirator/pkg/digdir/types"
	naisiov1 "github.com/nais/liberator/pkg/apis/nais.io/v1"
	"github.com/nais/liberator/pkg/kubernetes"
	"strings"
)

type FilteredScopeContainer struct {
	ExternalScopes []naisiov1.ExternalScope
	CurrentScopes  []Scope
	OwnedScopes    []types.ScopeRegistration
	ScopeToCreate  []naisiov1.ExternalScope
}

func NewFilterForScope(scope []naisiov1.ExternalScope) FilteredScopeContainer {
	return FilteredScopeContainer{
		ExternalScopes: scope,
		CurrentScopes:  []Scope{},
		OwnedScopes:    make([]types.ScopeRegistration, 0),
		ScopeToCreate:  make([]naisiov1.ExternalScope, 0),
	}
}

func (s FilteredScopeContainer) FilterScopes(actualScopesRegistrations []types.ScopeRegistration, desired clients.Instance) *FilteredScopeContainer {
	currentSubscopes := make([]string, 0)

	for _, externalScope := range s.ExternalScopes {
		scopeDoNotExist := true
		for _, actual := range actualScopesRegistrations {
			if scopeExistsInList(externalScope.Name, actual.Name) {
				scopeDoNotExist = false

				if scopeIsManaged(actual, desired) {
					if scopeMatches(actual, desired) {
						// match is found for already registered scope for team:namespace
						currentSubscopes = append(currentSubscopes, actual.Name)
						s.CurrentScopes = append(s.CurrentScopes, CreateScope(externalScope.Consumers, actual))
					}

					if scopeIsOwned(actual, desired) {
						s.OwnedScopes = append(s.OwnedScopes, actual)
					}
				}
			}
		}
		if scopeDoNotExist {
			// scope do not exist, will be created
			currentSubscopes = append(currentSubscopes, externalScope.Name)
			s.ScopeToCreate = append(s.ScopeToCreate, externalScope)
		}
	}

	// Setting current scopes for an app
	desired.GetStatus().SetApplicationScopes(currentSubscopes)
	return &s
}

func scopeExistsInList(externalScopeName string, actualScopeName string) bool {
	return externalScopeName == actualScopeName
}

func scopeMatches(actual types.ScopeRegistration, desired clients.Instance) bool {
	return actual.Description == kubernetes.UniformResourceScopeName(desired, actual.Name)
}

func splitUniqueName(description string) []string {
	uniqueName := strings.Split(description, ":")
	if len(uniqueName) < 2 {
		return nil
	}
	return strings.Split(description, ":")
}

func scopeIsManaged(actual types.ScopeRegistration, desired clients.Instance) bool {
	splitUniqueResourceName := splitUniqueName(actual.Description)
	if splitUniqueResourceName == nil {
		return false
	}
	cluster := splitUniqueResourceName[0]
	return cluster == desired.GetClusterName()
}

// scope is owned by a clustername:teamnamespace:scope
// scope can be used by several applications in a namespace(microservices), but only in that namespace
// To granulate the access an internal application can require a aud is set by the requesting application
func scopeIsOwned(actual types.ScopeRegistration, desired clients.Instance) bool {
	splitUniqueResourceName := splitUniqueName(actual.Description)
	if splitUniqueResourceName == nil {
		return false
	}
	namespace := splitUniqueResourceName[1]
	prefix := splitUniqueResourceName[2]
	return namespace != desired.GetNamespace() && prefix == clients.MaskinportenScopePrefix
}
