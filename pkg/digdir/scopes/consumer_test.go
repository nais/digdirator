package scopes

import (
	"github.com/nais/digdirator/pkg/digdir/types"
	"github.com/stretchr/testify/assert"
	"testing"
)

func TestConsumerFilteringForExistingConsumersInDifferentStates(t *testing.T) {

	// Preconditions

	// found
	// Consumer exists in either list, ExposedScope Consumer or Digdir ACL

	// swapped
	// false, we are looking for ExistingScope consumer list in Digdir ACL
	// true, we are looking consumer found in digdir ACL in ExistingScope consumer list

	// isActive
	// states if consumer should exist in digdir ACL or not

	consumer := CreateConsumer(false, types.StateDenied, "010101010")

	result := consumer.addOrUpdate(false, false, []Consumer{})
	consumer.ShouldBeAdded = true
	consumer.State = types.StateApproved
	assert.Equal(t, consumer, result[0], "Expect new consumer to be added to list with shouldBeAdded=true")

	consumer.State = types.StateDenied
	result = consumer.addOrUpdate(false, true, []Consumer{})
	assert.Equal(t, []Consumer{}, result, "Expect consumer NOT found actual consumer list but exists in digdir ACL and in DENIED state to be ignored")

	consumer.ShouldBeAdded = false
	consumer.State = types.StateApproved
	result = consumer.addOrUpdate(false, true, []Consumer{})
	assert.Equal(t, consumer, result[0], "Expect consumer existing in both list NOT in DENIED state should be kept, but not shouldBeAdded=false")

	consumer.ShouldBeAdded = true
	consumer.State = types.StateApproved
	result = consumer.addOrUpdate(true, true, []Consumer{})
	assert.Equal(t, []Consumer{}, result, "Expect consumer found in actual list but exist in digdir ACL to be activated again an added to list")
}
