package scopes

import (
	"github.com/nais/digdirator/pkg/clients"
	"github.com/nais/digdirator/pkg/digdir/types"
	naisiov1 "github.com/nais/liberator/pkg/apis/nais.io/v1"
	"github.com/stretchr/testify/assert"
	"testing"
	"time"
)

func TestConsumerFilteringWithAConsumerToRemove(t *testing.T) {

	expectedConsumerToRemove := "123456789"

	// Actual scope Consumer(s) are associated to
	scopeRegistrations := types.ScopeRegistration{
		Name:     "nav:test/scope",
		Subscope: "test/scope",
		Prefix:   "nav:",
	}

	// Consumers inn nais.yml and actual list of consumers
	exposedScopeConsumers := []naisiov1.ExposedScopeConsumer{
		{
			Name:  "Test",
			Orgno: "1010101010",
		},
	}

	// ACL Consumers registered at Maskinporten received from response
	consumerRegistrations := &[]types.ConsumerRegistration{
		{
			ConsumerOrgno: expectedConsumerToRemove,
			Created:       time.Time{},
			LastUpdated:   time.Time{},
			OwnerOrgno:    "000000000",
			Scope:         "nav:test/scope",
			State:         "APPROVED",
		},
		{
			ConsumerOrgno: "1010101010",
			Created:       time.Time{},
			LastUpdated:   time.Time{},
			OwnerOrgno:    "000000000",
			Scope:         "nav:test/scope",
			State:         "APPROVED",
		},
	}

	scope := CreateScope(exposedScopeConsumers, scopeRegistrations)

	_, result := scope.FilterConsumers(consumerRegistrations)

	assert.Equal(t, 1, len(result))
	assert.Equal(t, CreateConsumer(false, types.StateApproved, expectedConsumerToRemove), result[0])
}

func TestConsumerFilteringWithAConsumerToAdd(t *testing.T) {

	expectedConsumerToAdd := "123456789"

	// Actual scope Consumer(s) are associated to
	scopeRegistrations := types.ScopeRegistration{
		Name:     "nav:test/scope",
		Subscope: "test/scope",
		Prefix:   "nav:",
	}

	// Consumers inn nais.yml and actual list of consumers
	exposedScopeConsumers := []naisiov1.ExposedScopeConsumer{
		{
			Name:  "Test",
			Orgno: "1010101010",
		},
		{
			Name:  "Test2",
			Orgno: expectedConsumerToAdd,
		},
	}

	// ACL Consumers registered at Maskinporten received from response
	consumerRegistrations := &[]types.ConsumerRegistration{
		{
			ConsumerOrgno: "1010101010",
			Created:       time.Time{},
			LastUpdated:   time.Time{},
			OwnerOrgno:    "000000000",
			Scope:         "nav:test/scope",
			State:         "APPROVED",
		},
	}

	scope := CreateScope(exposedScopeConsumers, scopeRegistrations)

	_, result := scope.FilterConsumers(consumerRegistrations)

	assert.Equal(t, 1, len(result))
	assert.Equal(t, CreateConsumer(true, types.StateApproved, expectedConsumerToAdd), result[0])
}

func TestConsumerFilteringWithAConsumerToAddAndToRemove(t *testing.T) {

	expectedConsumerToAdd := "123456789"
	expectedConsumerToRemove := "1010101010"
	ignoredConsumerAlreadyInactiveInDeniedState := "11111111"
	consumerInDeniedStateActivatedAgain := "22222222"

	// Actual scope Consumer(s) are associated to
	scopeRegistrations := types.ScopeRegistration{
		Name:     "nav:test/scope",
		Subscope: "test/scope",
		Prefix:   "nav:",
	}

	// Consumers inn nais.yml and actual list of consumers
	exposedScopeConsumers := []naisiov1.ExposedScopeConsumer{
		{
			Name:  "Test2",
			Orgno: expectedConsumerToAdd,
		},
		{
			Name:  "Test3",
			Orgno: consumerInDeniedStateActivatedAgain,
		},
	}

	// ACL Consumers registered at Maskinporten received from response
	consumerRegistrations := &[]types.ConsumerRegistration{
		{
			ConsumerOrgno: expectedConsumerToRemove,
			Created:       time.Time{},
			LastUpdated:   time.Time{},
			OwnerOrgno:    "000000000",
			Scope:         "nav:test/scope",
			State:         "APPROVED",
		},
		// Should be ignored
		{
			ConsumerOrgno: ignoredConsumerAlreadyInactiveInDeniedState,
			Created:       time.Time{},
			LastUpdated:   time.Time{},
			OwnerOrgno:    "000000000",
			Scope:         "nav:test/scope",
			State:         "DENIED",
		},
		// Should be activated again
		{
			ConsumerOrgno: consumerInDeniedStateActivatedAgain,
			Created:       time.Time{},
			LastUpdated:   time.Time{},
			OwnerOrgno:    "000000000",
			Scope:         "nav:test/scope",
			State:         "DENIED",
		},
	}

	scope := CreateScope(exposedScopeConsumers, scopeRegistrations)

	_, result := scope.FilterConsumers(consumerRegistrations)

	assert.Equal(t, 3, len(result))
	assert.Equal(t, CreateConsumer(false, types.StateApproved, expectedConsumerToRemove), result[0])
	assert.Equal(t, CreateConsumer(true, types.StateApproved, expectedConsumerToAdd), result[1])
	assert.Equal(t, CreateConsumer(true, types.StateApproved, consumerInDeniedStateActivatedAgain), result[2])
}

func TestDesiredExposedScopeHasChanges(t *testing.T) {

	atMaxAge := 44
	maskinportenIntegration := []string{clients.MaskinportenDefaultAllowedIntegrationType}

	// Desired changes
	exposedScopes := []naisiov1.ExposedScope{
		{
			Name:                "KLP",
			AtAgeMax:            atMaxAge,
			AllowedIntegrations: maskinportenIntegration,
			Consumers:           []naisiov1.ExposedScopeConsumer{},
		},
	}

	// Actual scope in digdir
	scopeRegistrations := types.ScopeRegistration{
		Name:                   "nav:test/scope",
		Subscope:               "test/scope",
		Prefix:                 "nav:",
		AtMaxAge:               66,
		AllowedIntegrationType: maskinportenIntegration,
	}

	// AtAgeMax has changes
	scope := CreateScope([]naisiov1.ExposedScopeConsumer{}, scopeRegistrations)
	result := scope.HasChanged(exposedScopes)
	assert.True(t, result)

	// AllowedIntegrationType has changes
	scopeRegistrations.AtMaxAge = atMaxAge
	scopeRegistrations.AllowedIntegrationType = []string{"krr"}
	scope = CreateScope([]naisiov1.ExposedScopeConsumer{}, scopeRegistrations)
	result = scope.HasChanged(exposedScopes)
	assert.True(t, result)

	// No Changes
	scopeRegistrations.AtMaxAge = atMaxAge
	scopeRegistrations.AllowedIntegrationType = maskinportenIntegration
	scope = CreateScope([]naisiov1.ExposedScopeConsumer{}, scopeRegistrations)
	result = scope.HasChanged(exposedScopes)
	assert.False(t, result)
}
