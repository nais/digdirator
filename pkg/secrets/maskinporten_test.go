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

func TestMaskinportenStringData(t *testing.T) {
	client := maskinportenClient("test-name")

	jwk, err := crypto.GenerateJwk()
	assert.NoError(t, err)

	stringData, err := secrets.MaskinportenStringData(*jwk, client)
	assert.NoError(t, err, "should not error")

	t.Run("StringData should contain expected fields and values", func(t *testing.T) {
		t.Run("Secret Data should contain "+secrets.MaskinportenJwkKey, func(t *testing.T) {
			expected, err := json.Marshal(jwk)
			assert.NoError(t, err)
			assert.Equal(t, string(expected), stringData[secrets.MaskinportenJwkKey])
		})
		t.Run("Secret Data should contain "+secrets.MaskinportenWellKnownURL, func(t *testing.T) {
			expected := viper.GetString(config.DigDirMaskinportenBaseURL) + "/.well-known/oauth-authorization-server"
			assert.Equal(t, expected, stringData[secrets.MaskinportenWellKnownURL])
		})
		t.Run("Secret Data should contain "+secrets.MaskinportenClientID, func(t *testing.T) {
			assert.Equal(t, client.Status.ClientID, stringData[secrets.MaskinportenClientID])
		})
		t.Run("Secret Data should contain "+secrets.MaskinportenScopes+" with a single string of scopes separated by space", func(t *testing.T) {
			assert.Equal(t, secrets.JoinToString(client.Spec.Scopes), stringData[secrets.MaskinportenScopes])
		})
	})
}

func maskinportenClient(secretName string) *v1.MaskinportenClient {
	return &v1.MaskinportenClient{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test-app",
			Namespace: "test",
		},
		Spec: v1.MaskinportenClientSpec{
			SecretName: secretName,
			Scopes:     []string{"scope:one", "scope:two"},
		},
		Status: v1.ClientStatus{
			ClientID: "test-client-id",
		},
	}
}
