package v1

import (
	"github.com/nais/digdirator/pkg/digdir/types"
	"github.com/nais/digdirator/pkg/labels"
	"testing"

	"github.com/stretchr/testify/assert"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

const expectedIDPortenClientHash = "3f89fee23d842a44"

func TestIDPortenClient_MakeDescription(t *testing.T) {
	expected := "test-cluster:test-namespace:test-app"
	assert.Equal(t, expected, minimalIDPortenClient().MakeDescription())
}

func TestIDPortenClient_CalculateHash(t *testing.T) {
	actual, err := minimalIDPortenClient().CalculateHash()
	assert.NoError(t, err)
	assert.Equal(t, expectedIDPortenClientHash, actual)
}

func TestIDPortenClient_MakeLabels(t *testing.T) {
	app := minimalIDPortenClient()
	expected := labels.IDPortenLabels(app)
	assert.Equal(t, app.MakeLabels(), expected)
}

func TestIDPortenClient_IsHashUnchanged(t *testing.T) {
	t.Run("Minimal Application should have unchanged hash value", func(t *testing.T) {
		actual, err := minimalIDPortenClient().IsHashUnchanged()
		assert.NoError(t, err)
		assert.True(t, actual)
	})
	t.Run("Application with changed value should have changed hash value", func(t *testing.T) {
		app := minimalIDPortenClient()
		app.Spec.ClientURI = "changed"
		actual, err := app.IsHashUnchanged()
		assert.NoError(t, err)
		assert.False(t, actual)
	})
}

func TestIDPortenClient_SetHash(t *testing.T) {
	app := minimalIDPortenClient()
	app.Spec.ClientURI = "changed"

	hash, err := app.CalculateHash()
	assert.NoError(t, err)
	app.GetStatus().SetHash(hash)
	assert.Equal(t, "f409103c18d3017b", app.GetStatus().GetHash())
}

func TestIDPortenClient_IntegrationType(t *testing.T) {
	app := minimalIDPortenClient()
	assert.Equal(t, types.IntegrationTypeIDPorten, app.GetIntegrationType())
}

func minimalIDPortenClient() *IDPortenClient {
	return &IDPortenClient{
		ObjectMeta: metav1.ObjectMeta{
			Name:        "test-app",
			Namespace:   "test-namespace",
			ClusterName: "test-cluster",
		},
		Spec: IDPortenClientSpec{
			ClientURI:              "test",
			RedirectURI:            "https://test.com",
			SecretName:             "test",
			FrontchannelLogoutURI:  "test",
			PostLogoutRedirectURIs: nil,
			RefreshTokenLifetime:   3600,
		},
		Status: ClientStatus{
			ProvisionHash: expectedIDPortenClientHash,
		},
	}
}
