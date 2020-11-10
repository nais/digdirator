package v1

import (
	"github.com/nais/digdirator/pkg/digdir/types"
	"testing"

	"github.com/stretchr/testify/assert"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

var finalizerName = "test-finalizer"

const expectedHash = "3f89fee23d842a44"

func TestIDPortenClient_GetUniqueName(t *testing.T) {
	expected := "test-cluster:test-namespace:test-app"
	assert.Equal(t, expected, minimalClient().GetUniqueName())
}

func TestIDPortenClient_Hash(t *testing.T) {
	actual, err := minimalClient().Hash()
	assert.NoError(t, err)
	assert.Equal(t, expectedHash, actual)
}

func TestIDPortenClient_HashUnchanged(t *testing.T) {
	t.Run("Minimal Application should have unchanged hash value", func(t *testing.T) {
		actual, err := minimalClient().HashUnchanged()
		assert.NoError(t, err)
		assert.True(t, actual)
	})
	t.Run("Application with changed value should have changed hash value", func(t *testing.T) {
		app := minimalClient()
		app.Spec.ClientURI = "changed"
		actual, err := app.HashUnchanged()
		assert.NoError(t, err)
		assert.False(t, actual)
	})
}

func TestIDPortenClient_UpdateHash(t *testing.T) {
	app := minimalClient()
	app.Spec.ClientURI = "changed"

	err := app.UpdateHash()
	assert.NoError(t, err)
	assert.Equal(t, "f409103c18d3017b", app.Status.ProvisionHash)
}

func TestIDPortenClient_IntegrationType(t *testing.T) {
	app := minimalClient()
	assert.Equal(t, types.IntegrationTypeIDPorten, app.IntegrationType())
}

func minimalClient() *IDPortenClient {
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
			ProvisionHash: expectedHash,
		},
	}
}
