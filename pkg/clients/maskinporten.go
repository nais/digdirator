package clients

import (
	"github.com/nais/digdirator/pkg/digdir/types"
	v1 "github.com/nais/liberator/pkg/apis/nais.io/v1"
)

type FilteredScopes struct {
	Valid   []string
	Invalid []string
}

func FilterScopes(desired []v1.MaskinportenScope, accesibleScopes []types.Scope) FilteredScopes {
	validScopes := make([]string, 0)
	invalidScopes := make([]string, 0)

	for _, scope := range desired {
		if scopeIsAccessible(scope, accesibleScopes) {
			validScopes = append(validScopes, scope.Name)
		} else {
			invalidScopes = append(invalidScopes, scope.Name)
		}
	}

	return FilteredScopes{
		Valid:   validScopes,
		Invalid: invalidScopes,
	}
}

func scopeIsAccessible(scope v1.MaskinportenScope, accessibleScopes []types.Scope) bool {
	for _, accessible := range accessibleScopes {
		if scope.Name == accessible.Scope && accessible.State == types.ScopeAccessApproved {
			return true
		}
	}
	return false
}
