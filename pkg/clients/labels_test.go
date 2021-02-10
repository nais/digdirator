package clients_test

import (
	"github.com/nais/digdirator/pkg/clients"
	"github.com/stretchr/testify/assert"
	"testing"
)

func TestMakeLabels_IDPortenClient(t *testing.T) {
	client := minimalIDPortenClient()

	actual := clients.MakeLabels(client)

	assert.Equal(t, map[string]string{
		clients.AppLabelKey:  client.GetName(),
		clients.TypeLabelKey: clients.IDPortenTypeLabelValue,
	}, actual)
}

func TestMakeLabels_MaskinportenClient(t *testing.T) {
	client := minimalMaskinportenClient()

	actual := clients.MakeLabels(client)

	assert.Equal(t, map[string]string{
		clients.AppLabelKey:  client.GetName(),
		clients.TypeLabelKey: clients.MaskinportenTypeLabelValue,
	}, actual)
}
