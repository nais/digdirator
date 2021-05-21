package scopes

import (
	"fmt"
	"github.com/nais/digdirator/pkg/clients"
	"github.com/nais/digdirator/pkg/digdir/types"
	naisiov1 "github.com/nais/liberator/pkg/apis/nais.io/v1"
	"github.com/stretchr/testify/assert"
	"testing"
	"time"
)

func TestConsumerFilteringWithNoChanges(t *testing.T) {

	expectedConsumer := "1010101010"
	subscope := "test/scope"
	name := fmt.Sprintf("nav:%s", subscope)

	// Consumers inn nais.yml and actual list of consumers
	exposedScopeConsumers := createExposedConsumers(expectedConsumer)

	// ACL Consumers registered at Maskinporten received from response
	consumerRegistrations := createConsumerRegistrations(expectedConsumer)

	// Actual scope Consumer(s) are associated to
	// create scope struct
	scope := CurrentScopeInfo(createScopeRegistrations(name, subscope)[0], createExposedScope(exposedScopeConsumers, subscope)[0])

	consumerStatus, filteredResult := scope.FilterConsumers(&consumerRegistrations)

	assert.Equal(t, 0, len(filteredResult))
	assert.Equal(t, 1, len(consumerStatus))
	assert.Equal(t, expectedConsumer, consumerStatus[0])
}

func TestConsumerFilteringWithAConsumerToRemove(t *testing.T) {

	expectedConsumerToRemove := "123456789"
	existingConsumer := "1010101010"
	subscope := "test/scope"
	name := fmt.Sprintf("nav:%s", subscope)

	// Existing consumer
	exposedScopeConsumers := createExposedConsumers(existingConsumer)
	// existing consumer and consumer to remove
	consumerRegistrations := createConsumerRegistrations(existingConsumer, expectedConsumerToRemove)
	scope := CurrentScopeInfo(createScopeRegistrations(name, subscope)[0], createExposedScope(exposedScopeConsumers, subscope)[0])

	consumerStatus, result := scope.FilterConsumers(&consumerRegistrations)

	assert.Equal(t, 1, len(result))
	assert.Equal(t, CreateConsumer(false, types.ScopeStateApproved, expectedConsumerToRemove), result[0])
	assert.Equal(t, 1, len(consumerStatus))
	assert.Equal(t, existingConsumer, consumerStatus[0])
}

func TestConsumerFilteringWithAConsumerToAdd(t *testing.T) {

	expectedConsumerToAdd := "123456789"
	existingConsumer := "1010101010"
	subscope := "test/scope"
	name := fmt.Sprintf("nav:%s", subscope)

	// existing and consumer to add
	exposedScopeConsumers := createExposedConsumers(existingConsumer, expectedConsumerToAdd)
	// only existing consumer
	consumerRegistrations := createConsumerRegistrations(existingConsumer)
	scope := CurrentScopeInfo(createScopeRegistrations(name, subscope)[0], createExposedScope(exposedScopeConsumers, subscope)[0])
	consumerStatus, result := scope.FilterConsumers(&consumerRegistrations)

	assert.Equal(t, 1, len(result))
	assert.Equal(t, CreateConsumer(true, types.ScopeStateApproved, expectedConsumerToAdd), result[0])
	assert.Equal(t, 1, len(consumerStatus))
	assert.Equal(t, existingConsumer, consumerStatus[0])
	// consumer added recently get added to consumerStatus list after remote/digdir is updated with new consumer in acl
}

func TestConsumerFilteringWithAConsumerToAddAndToRemoveToActivateAndExisting(t *testing.T) {

	// More complex test
	expectedConsumerToAdd := "123456789"
	existingConsumer := "923456781"
	expectedConsumerToRemove := "1010101010"
	ignoredConsumerAlreadyInactiveInDeniedState := "11111111"
	consumerInDeniedStateActivatedAgain := "22222222"
	subscope := "test/scope"
	name := fmt.Sprintf("nav:%s", subscope)

	exposedScopeConsumers := createExposedConsumers(expectedConsumerToAdd, consumerInDeniedStateActivatedAgain, existingConsumer)

	consumerRegistrations := &[]types.ConsumerRegistration{
		// Existing, keep as is
		{
			ConsumerOrgno: existingConsumer,
			Created:       time.Time{},
			LastUpdated:   time.Time{},
			OwnerOrgno:    "000000000",
			Scope:         "nav:test/scope",
			State:         "APPROVED",
		},
		// Should be removed
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

	scope := CurrentScopeInfo(createScopeRegistrations(name, subscope)[0], createExposedScope(exposedScopeConsumers, subscope)[0])

	consumerStatus, result := scope.FilterConsumers(consumerRegistrations)

	assert.Equal(t, 3, len(result))

	validConsumers := map[string]Consumer{
		expectedConsumerToRemove:            CreateConsumer(false, types.ScopeStateApproved, expectedConsumerToRemove),
		expectedConsumerToAdd:               CreateConsumer(true, types.ScopeStateApproved, expectedConsumerToAdd),
		consumerInDeniedStateActivatedAgain: CreateConsumer(true, types.ScopeStateApproved, consumerInDeniedStateActivatedAgain),
	}

	for _, v := range result {
		if _, ok := validConsumers[v.Orgno]; ok {
			assert.True(t, ok, "Map should valid consumer, either getting updated or deleted")
		}
	}

	assert.Equal(t, 2, len(consumerStatus))
	for _, v := range consumerStatus {
		if v == consumerInDeniedStateActivatedAgain {
			assert.Equal(t, consumerInDeniedStateActivatedAgain, v)
		}
		if v == existingConsumer {
			assert.Equal(t, existingConsumer, v)
		}
	}
}

func TestDesiredExposedScopeHasChanges(t *testing.T) {

	maskinportenIntegration := []string{clients.MaskinportenDefaultAllowedIntegrationType}
	product := "arbeid"

	// Actual scope in digdir
	scopeRegistrations := types.ScopeRegistration{
		Name:                   fmt.Sprintf("nav:%s/test/scope", product),
		Subscope:               fmt.Sprintf("%s/test/scope", product),
		Prefix:                 "nav:",
		AtMaxAge:               30,
		AllowedIntegrationType: maskinportenIntegration,
	}

	exposedScope := naisiov1.ExposedScope{
		Enabled:   true,
		Name:      scopeRegistrations.Subscope,
		Product:   product,
		Consumers: []naisiov1.ExposedScopeConsumer{},
	}

	// No changes - default values is configured for costume val
	scope := CurrentScopeInfo(scopeRegistrations, exposedScope)
	result := scope.HasChanged()
	assert.False(t, result)

	// AtAgeMax has changes
	exposedScope.AtMaxAge = 33
	scope = CurrentScopeInfo(scopeRegistrations, exposedScope)
	result = scope.HasChanged()
	assert.True(t, result)

	// AllowedIntegrationType has changes
	scopeRegistrations.AllowedIntegrationType = []string{"krr"}
	scope = CurrentScopeInfo(scopeRegistrations, exposedScope)
	result = scope.HasChanged()
	assert.True(t, result)
}

func createScopeRegistrations(name, subscope string) []types.ScopeRegistration {
	return []types.ScopeRegistration{
		{
			Name:     name,
			Subscope: subscope,
			Prefix:   "nav:",
		},
	}
}

func createExposedConsumers(consumers ...string) []naisiov1.ExposedScopeConsumer {
	exposed := make([]naisiov1.ExposedScopeConsumer, 0)
	for i, c := range consumers {
		exposed = append(exposed, naisiov1.ExposedScopeConsumer{
			Name:  fmt.Sprintf("test%d", i),
			Orgno: c,
		})
	}
	return exposed
}

func createExposedScope(exposedConsumers []naisiov1.ExposedScopeConsumer, subscopes ...string) []naisiov1.ExposedScope {
	exposed := make([]naisiov1.ExposedScope, 0)
	for _, s := range subscopes {
		exposed = append(exposed, naisiov1.ExposedScope{
			Enabled:             true,
			Name:                s,
			AtMaxAge:            30,
			AllowedIntegrations: []string{"maskinporten"},
			Consumers:           exposedConsumers,
		})
	}
	return exposed
}

func createConsumerRegistrations(consumers ...string) []types.ConsumerRegistration {
	exposed := make([]types.ConsumerRegistration, 0)
	for _, c := range consumers {
		exposed = append(exposed, types.ConsumerRegistration{
			ConsumerOrgno: c,
			Created:       time.Time{},
			LastUpdated:   time.Time{},
			OwnerOrgno:    "000000000",
			Scope:         "nav:test/scope",
			State:         "APPROVED",
		})
	}
	return exposed
}
