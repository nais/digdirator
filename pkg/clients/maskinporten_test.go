package clients_test

import (
	"github.com/nais/digdirator/pkg/clients"
	"github.com/nais/digdirator/pkg/digdir/types"
	v1 "github.com/nais/liberator/pkg/apis/nais.io/v1"
	"github.com/stretchr/testify/assert"
	"testing"
)

func TestFilterScopes(t *testing.T) {
	desired := v1.MaskinportenScope{
		ConsumedScopes: []v1.ConsumedScope{
			{
				Name: "valid-scope",
			},
			{
				Name: "invalid-scope",
			},
			{
				Name: "not-found-scope",
			},
		},
	}

	accessibleScopes := []types.Scope{
		{
			ConsumerOrgNo: "our-org-no",
			OwnerOrgNo:    "their-org-no",
			Scope:         "valid-scope",
			State:         types.ScopeAccessApproved,
		},
		{
			ConsumerOrgNo: "our-org-no",
			OwnerOrgNo:    "their-org-no",
			Scope:         "invalid-scope",
			State:         types.ScopeAccessDenied,
		},
	}

	filteredScopes := clients.FilterScopes(desired.ConsumedScopes, accessibleScopes)

	assert.Len(t, filteredScopes.Valid, 1)
	assert.Contains(t, filteredScopes.Valid, "valid-scope")

	assert.Len(t, filteredScopes.Invalid, 2)
	assert.Contains(t, filteredScopes.Invalid, "invalid-scope")
	assert.Contains(t, filteredScopes.Invalid, "not-found-scope")
}
