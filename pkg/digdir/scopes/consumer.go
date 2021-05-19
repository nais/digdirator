package scopes

import (
	"github.com/nais/digdirator/pkg/digdir/types"
)

type Consumer struct {
	ShouldBeAdded bool
	State         types.State
	Orgno         string
}

func CreateConsumer(shouldBeAdded bool, state types.State, orgno string) Consumer {
	return Consumer{
		ShouldBeAdded: shouldBeAdded,
		State:         state,
		Orgno:         orgno,
	}
}

func (c Consumer) findIn(consumers map[string]Consumer) bool {
	found := false
	for _, consumer := range consumers {
		if consumer.Orgno == c.Orgno {
			found = true
			break
		}
	}
	return found
}

func (c Consumer) addOrUpdate(found, swapped bool, consumerList []Consumer) []Consumer {

	// Consumer is not in digdir acl
	if !found {
		// Compare exposedConsumer against digdir acl
		if !swapped {
			c.ShouldBeAdded = true
			c.State = types.StateApproved
			consumerList = append(consumerList, c)
		}
		// Keep existing, ignore consumers in denied state
		if swapped && c.State != types.StateDenied {
			consumerList = append(consumerList, c)
		}
	}

	// Consumer found, check digidir response acl against consumer list to re-activate denied consumer
	if found && swapped && c.State == types.StateDenied {
		c.ShouldBeAdded = true
		c.State = types.StateApproved
		consumerList = append(consumerList, c)
	}
	return consumerList
}
