package clients_test

import (
	"testing"

	"github.com/nais/digdirator/pkg/clients"
	"github.com/nais/digdirator/pkg/fixtures"

	"github.com/stretchr/testify/assert"
)

func TestMakeLabels_IDPortenClient(t *testing.T) {
	client := fixtures.MinimalIDPortenClient()

	actual := clients.MakeLabels(client)

	assert.Equal(t, map[string]string{
		clients.AppLabelKey:  client.GetName(),
		clients.TypeLabelKey: clients.IDPortenTypeLabelValue,
	}, actual)
}

func TestMakeLabels_MaskinportenClient(t *testing.T) {
	client := fixtures.MinimalMaskinportenClient()

	actual := clients.MakeLabels(client)

	assert.Equal(t, map[string]string{
		clients.AppLabelKey:  client.GetName(),
		clients.TypeLabelKey: clients.MaskinportenTypeLabelValue,
	}, actual)
}
