package scopes

import (
	"fmt"
	"github.com/nais/digdirator/pkg/clients"
	"github.com/nais/digdirator/pkg/digdir/types"
	naisiov1 "github.com/nais/liberator/pkg/apis/nais.io/v1"
)

const NumberOfPermutation = 2

type Scope struct {
	ScopeRegistration types.ScopeRegistration
	CurrentScope      naisiov1.ExposedScope
}

func CurrentScopeInfo(registration types.ScopeRegistration, scope naisiov1.ExposedScope) Scope {
	return Scope{
		ScopeRegistration: registration,
		CurrentScope:      scope,
	}
}

func (s Scope) FilterConsumers(acl *[]types.ConsumerRegistration) ([]string, []Consumer) {
	differance := make([]Consumer, 0)
	aclConsumerList := toConsumers(acl)
	expectedConsumerList := s.toConsumers()

	consumerStatus := make([]string, 0)

	swapped := false
	for i := 0; i < NumberOfPermutation; i++ {
		for _, consumer := range expectedConsumerList {

			found := consumer.findIn(aclConsumerList)
			addConsumerStatus(found, swapped, consumer.Orgno, &consumerStatus)
			differance = consumer.addOrUpdate(found, swapped, differance)
		}
		// Swapping
		if i == 0 {
			expectedConsumerList, aclConsumerList = aclConsumerList, expectedConsumerList
			swapped = true
		}
	}
	return consumerStatus, differance
}

func addConsumerStatus(found, swapped bool, consumerOrgno string, consumerStatus *[]string) {
	if found && !swapped {
		*consumerStatus = append(*consumerStatus, consumerOrgno)
	}
}

func (s Scope) toConsumers() map[string]Consumer {
	consumers := make(map[string]Consumer)
	for _, consumer := range s.CurrentScope.Consumers {
		consumers[consumer.Orgno] = CreateConsumer(false, types.ScopeStateDenied, consumer.Orgno)
	}
	return consumers
}

func toConsumers(acl *[]types.ConsumerRegistration) map[string]Consumer {
	consumers := make(map[string]Consumer)
	for _, consumer := range *acl {
		consumers[consumer.ConsumerOrgno] = CreateConsumer(false, consumer.State, consumer.ConsumerOrgno)
	}
	return consumers
}

func (s Scope) ToString() string {
	return fmt.Sprintf("%s:%s", s.ScopeRegistration.Prefix, s.ScopeRegistration.Subscope)
}

func (s Scope) HasChanged() bool {
	clients.SetDefaultScopeValues(&s.CurrentScope)
	switch {
	case s.ScopeRegistration.AtMaxAge != *s.CurrentScope.AtMaxAge:
		return true
	case !equals(s.ScopeRegistration.AllowedIntegrationType, s.CurrentScope.AllowedIntegrations):
		return true
	}
	return false
}

// IsActive exposed scope should be active or not
func (s Scope) IsActive() bool {
	return s.CurrentScope.Enabled
}

// CanBeActivated Existing and inactive scope need to be activated again
func (s Scope) CanBeActivated() bool {
	return s.IsActive() && !s.ScopeRegistration.Active
}

func equals(actual, desired []string) bool {
	if len(actual) != len(desired) {
		return false
	}
	for i, value := range desired {
		if value != actual[i] {
			return false
		}
	}
	return true
}
