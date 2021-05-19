package scopes

import (
	"fmt"
	"github.com/nais/digdirator/pkg/digdir/types"
	naisiov1 "github.com/nais/liberator/pkg/apis/nais.io/v1"
	"github.com/nais/liberator/pkg/kubernetes"
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
		consumers[consumer.Orgno] = CreateConsumer(false, types.StateDenied, consumer.Orgno)
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

func (s Scope) HasChanged(desired map[string]naisiov1.ExposedScope) bool {
	for d, scope := range desired {
		if kubernetes.ToScope(scope.Product, d) == s.ScopeRegistration.Subscope {
			switch {
			case scope.AtAgeMax != 0 && s.ScopeRegistration.AtMaxAge != scope.AtAgeMax:
				return true
			case !equals(s.ScopeRegistration.AllowedIntegrationType, scope.AllowedIntegrations):
				return true
			}
		}
	}
	return false
}

// IsActive exposed scope should be active or not
func (s Scope) IsActive(desired map[string]naisiov1.ExposedScope) bool {
	scope, err := s.GetExposedScope(desired)
	if err == nil {
		return scope.Enabled
	}
	return false
}

func (s Scope) GetExposedScope(desired map[string]naisiov1.ExposedScope) (naisiov1.ExposedScope, error) {
	for d, scope := range desired {
		if kubernetes.ToScope(scope.Product, d) == s.ScopeRegistration.Subscope {
			return scope, nil
		}
	}
	return naisiov1.ExposedScope{}, fmt.Errorf("could not find scope")
}

// CanBeActivated Existing and inactive scope need to be activated again
func (s Scope) CanBeActivated(desired map[string]naisiov1.ExposedScope) bool {
	return s.IsActive(desired) && !s.ScopeRegistration.Active
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
