package scopes

import (
	naisiov1 "github.com/nais/liberator/pkg/apis/nais.io/v1"
	"github.com/nais/liberator/pkg/kubernetes"

	"github.com/nais/digdirator/pkg/digdir/types"
)

type Operations struct {
	ToCreate []naisiov1.ExposedScope
	ToUpdate []Scope
}

func Generate(actualScopes []types.ScopeRegistration, desiredScopes map[string]naisiov1.ExposedScope) *Operations {
	ops := &Operations{
		ToCreate: make([]naisiov1.ExposedScope, 0),
		ToUpdate: make([]Scope, 0),
	}

	subscopes := make(map[string]types.ScopeRegistration)
	for _, actual := range actualScopes {
		subscopes[actual.Subscope] = actual
	}

	for _, desired := range desiredScopes {
		desiredSubscope := kubernetes.ToScope(desired.Product, desired.Name)

		subscope, ok := subscopes[desiredSubscope]
		if ok {
			ops.ToUpdate = append(ops.ToUpdate, CurrentScopeInfo(subscope, desired))
		} else {
			ops.ToCreate = append(ops.ToCreate, desired)
		}
	}

	return ops
}
