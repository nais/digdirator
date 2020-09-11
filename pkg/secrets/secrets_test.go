package secrets_test

import (
	"encoding/json"
	"github.com/nais/digdirator/pkg/config"
	"github.com/nais/digdirator/pkg/crypto"
	"github.com/nais/digdirator/pkg/secrets"
	"github.com/spf13/viper"
	"testing"

	v1 "github.com/nais/digdirator/api/v1"
	"github.com/nais/digdirator/pkg/labels"
	"github.com/stretchr/testify/assert"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func TestCreateSecretSpec(t *testing.T) {
	client := &v1.IDPortenClient{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test-app",
			Namespace: "test",
		},
		Spec: v1.IDPortenClientSpec{
			SecretName:  "test-secret",
			RedirectURI: "https://my-app.nav.no",
		},
		Status: v1.IDPortenClientStatus{
			ClientID: "test-client-id",
		},
	}
	jwk, err := crypto.GenerateJwk()
	assert.NoError(t, err)

	spec, err := secrets.Spec(client, *jwk)
	assert.NoError(t, err, "should not error")

	stringData, err := secrets.StringData(*jwk, client)
	assert.NoError(t, err, "should not error")

	t.Run("Name should equal provided name in Spec", func(t *testing.T) {
		expected := client.Spec.SecretName
		actual := spec.Name
		assert.NotEmpty(t, actual)
		assert.Equal(t, expected, actual)
	})

	t.Run("Secret spec should be as expected", func(t *testing.T) {
		expected := &corev1.Secret{
			TypeMeta: metav1.TypeMeta{
				Kind:       "Secret",
				APIVersion: "v1",
			},
			ObjectMeta: secrets.ObjectMeta(client),
			StringData: stringData,
			Type:       corev1.SecretTypeOpaque,
		}
		assert.NotEmpty(t, spec)
		assert.Equal(t, expected, spec)

		assert.Equal(t, corev1.SecretTypeOpaque, spec.Type, "Secret Type should be Opaque")
	})

	t.Run("StringData should contain expected fields and values", func(t *testing.T) {
		t.Run("Secret Data should contain Private JWK", func(t *testing.T) {
			expected, err := json.Marshal(jwk)
			assert.NoError(t, err)
			assert.Equal(t, string(expected), spec.StringData[secrets.JwkKey])
		})
		t.Run("Secret Data should contain well-known URL", func(t *testing.T) {
			expected := viper.GetString(config.DigDirAuthBaseURL) + "/idporten-oidc-provider/.well-known/openid-configuration"
			assert.Equal(t, expected, spec.StringData[secrets.WellKnownURL])
		})
		t.Run("Secret Data should contain client ID", func(t *testing.T) {
			assert.Equal(t, client.Status.ClientID, spec.StringData[secrets.ClientID])
		})
		t.Run("Secret Data should contain redirect URI", func(t *testing.T) {
			assert.Equal(t, client.Spec.RedirectURI, spec.StringData[secrets.RedirectURI])
		})
	})
}

func TestObjectMeta(t *testing.T) {
	name := "test-name"
	app := &v1.IDPortenClient{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test-app",
			Namespace: "test",
		},
		Spec: v1.IDPortenClientSpec{
			SecretName: name,
		},
	}

	om := secrets.ObjectMeta(app)

	t.Run("Name should be set", func(t *testing.T) {
		actual := om.GetName()
		assert.NotEmpty(t, actual)
		assert.Equal(t, name, actual)
	})

	t.Run("Namespace should be set", func(t *testing.T) {
		actual := om.GetNamespace()
		assert.NotEmpty(t, actual)
		assert.Equal(t, app.GetNamespace(), actual)
	})
	t.Run("Labels should be set", func(t *testing.T) {
		actualLabels := om.GetLabels()
		expectedLabels := map[string]string{
			labels.AppLabelKey:  app.GetName(),
			labels.TypeLabelKey: labels.TypeLabelValue,
		}
		assert.NotEmpty(t, actualLabels, "Labels should not be empty")
		assert.Equal(t, expectedLabels, actualLabels, "Labels should be set")
	})
}
