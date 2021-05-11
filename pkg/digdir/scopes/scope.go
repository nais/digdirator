package scopes

import (
	"fmt"
	"github.com/nais/digdirator/pkg/digdir/types"
	naisiov1 "github.com/nais/liberator/pkg/apis/nais.io/v1"
	"sort"
)

const NumberOfPermutation = 2

type Scope struct {
	Consumers         []naisiov1.ExposedScopeConsumer
	ScopeRegistration types.ScopeRegistration
}

func CreateScope(consumers []naisiov1.ExposedScopeConsumer, registration types.ScopeRegistration) Scope {
	return Scope{
		Consumers:         consumers,
		ScopeRegistration: registration,
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
	sortConsumers(differance)
	return consumerStatus, differance
}

func addConsumerStatus(found, swapped bool, consumerOrgno string, consumerStatus *[]string) {
	if found && !swapped {
		*consumerStatus = append(*consumerStatus, consumerOrgno)
	}
}

func (s Scope) toConsumers() map[string]Consumer {
	consumers := make(map[string]Consumer)
	for _, consumer := range s.Consumers {
		consumers[consumer.Orgno] = CreateConsumer(false, types.StateApproved, consumer.Orgno)
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

func sortConsumers(consumerList []Consumer) {
	sort.SliceStable(consumerList, func(i, j int) bool {
		return consumerList[i].Orgno < consumerList[j].Orgno
	})
}

func (s Scope) HasChanged(desires []naisiov1.ExposedScope) bool {
	change := false
	for _, desired := range desires {
		switch {
		case s.ScopeRegistration.AtMaxAge != desired.AtAgeMax:
			change = true
		case !equals(s.ScopeRegistration.AllowedIntegrationType, desired.AllowedIntegrations):
			change = true
		}
	}
	return change
}

func equals(actual, desires []string) bool {
	if len(actual) != len(desires) {
		return false
	}
	for i, value := range actual {
		if value != desires[i] {
			return false
		}
	}
	return true
}
