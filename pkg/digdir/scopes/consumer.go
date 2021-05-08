package scopes

type Consumer struct {
	Active bool
	Orgno  string
}

func CreateConsumer(shouldBeAdded bool, orgno string) Consumer {
	return Consumer{
		Active: shouldBeAdded,
		Orgno:  orgno,
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
	if !found {
		if !swapped {
			c.Active = true
			consumerList = append(consumerList, c)
		} else {
			consumerList = append(consumerList, c)
		}
	}
	return consumerList
}
