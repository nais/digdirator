package secrets_test

import (
	"encoding/json"
	v1 "github.com/nais/digdirator/api/v1"
	"github.com/nais/digdirator/pkg/config"
	"github.com/nais/digdirator/pkg/crypto"
	"github.com/nais/digdirator/pkg/secrets"
	"github.com/spf13/viper"
	"github.com/stretchr/testify/assert"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"testing"
)

func TestIDPortenStringData(t *testing.T) {
	client := idPortenClient("test-name")

	jwk, err := crypto.GenerateJwk()
	assert.NoError(t, err)

	stringData, err := secrets.IDPortenStringData(*jwk, client)
	assert.NoError(t, err, "should not error")

	t.Run("StringData should contain expected fields and values", func(t *testing.T) {
		t.Run("Secret Data should contain "+secrets.IDPortenJwkKey, func(t *testing.T) {
			expected, err := json.Marshal(jwk)
			assert.NoError(t, err)
			assert.Equal(t, string(expected), stringData[secrets.IDPortenJwkKey])
		})
		t.Run("Secret Data should contain "+secrets.IDPortenWellKnownURL, func(t *testing.T) {
			expected := viper.GetString(config.DigDirAuthBaseURL) + "/idporten-oidc-provider/.well-known/openid-configuration"
			assert.Equal(t, expected, stringData[secrets.IDPortenWellKnownURL])
		})
		t.Run("Secret Data should contain "+secrets.IDPortenClientID, func(t *testing.T) {
			assert.Equal(t, client.Status.ClientID, stringData[secrets.IDPortenClientID])
		})
		t.Run("Secret Data should contain "+secrets.IDPortenRedirectURI, func(t *testing.T) {
			assert.Equal(t, client.Spec.RedirectURI, stringData[secrets.IDPortenRedirectURI])
		})
	})
}

func idPortenClient(secretName string) *v1.IDPortenClient {
	return &v1.IDPortenClient{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test-app",
			Namespace: "test",
		},
		Spec: v1.IDPortenClientSpec{
			SecretName:  secretName,
			RedirectURI: "https://my-app.nav.no",
		},
		Status: v1.ClientStatus{
			ClientID: "test-client-id",
		},
	}
}
