package scopes

import (
	"fmt"
	"strings"

	naisiov1 "github.com/nais/liberator/pkg/apis/nais.io/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/nais/digdirator/pkg/digdir/types"
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

// Is here for feature ref. tracking all consumers for a scope
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

// Description generates the Maskinporten scope description.
// TODO: this should ideally be configurable in the future Scope resource.
func Description(resource metav1.Object, clusterName, product string) string {
	team := resource.GetNamespace()
	app := resource.GetName()

	return fmt.Sprintf("%s - %s:%s:%s", product, clusterName, team, app)
}

// Subscope generates the Maskinporten subscope name.
// If the separator is empty, it will default to ":" if the name contains a "/", otherwise it will default to ":".
// Format: `<product><separator><name>`
func Subscope(exposedScope naisiov1.ExposedScope) string {
	product := exposedScope.Product
	name := exposedScope.Name

	separator := ""
	if exposedScope.Separator != nil {
		separator = *exposedScope.Separator
	}

	if separator == "" {
		// TODO: The original comment for this logic was "able to use legacy scopes from on-prem in gcp", but that doesn't really make any sense.
		//  No one seems to know why this is needed, but we can't change this without breaking existing scope definitions.
		//  We should remove the logic when we migrate to the new separate Scope resource.
		separator = ":"
		if strings.Contains(name, "/") {
			separator = "/"
		}
	}

	return product + separator + name
}
