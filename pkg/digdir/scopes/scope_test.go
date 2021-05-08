package scopes

import (
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
	externalScopeConsumers := []naisiov1.ExternalScopeConsumer{
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

	scope := CreateScope(externalScopeConsumers, scopeRegistrations)

	_, result := scope.FilterConsumers(consumerRegistrations)

	assert.Equal(t, 1, len(result))
	assert.Equal(t, CreateConsumer(false, expectedConsumerToRemove), result[0])
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
	externalScopeConsumers := []naisiov1.ExternalScopeConsumer{
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

	scope := CreateScope(externalScopeConsumers, scopeRegistrations)

	_, result := scope.FilterConsumers(consumerRegistrations)

	assert.Equal(t, 1, len(result))
	assert.Equal(t, CreateConsumer(true, expectedConsumerToAdd), result[0])
}

func TestConsumerFilteringWithAConsumerToAddAndToRemove(t *testing.T) {

	expectedConsumerToAdd := "123456789"
	expectedConsumerToRemove := "1010101010"

	// Actual scope Consumer(s) are associated to
	scopeRegistrations := types.ScopeRegistration{
		Name:     "nav:test/scope",
		Subscope: "test/scope",
		Prefix:   "nav:",
	}

	// Consumers inn nais.yml and actual list of consumers
	externalScopeConsumers := []naisiov1.ExternalScopeConsumer{
		{
			Name:  "Test2",
			Orgno: expectedConsumerToAdd,
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
	}

	scope := CreateScope(externalScopeConsumers, scopeRegistrations)

	_, result := scope.FilterConsumers(consumerRegistrations)

	assert.Equal(t, 2, len(result))
	assert.Equal(t, CreateConsumer(true, expectedConsumerToAdd), result[0])
	assert.Equal(t, CreateConsumer(false, expectedConsumerToRemove), result[1])
}
