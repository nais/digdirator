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

func TestCreateSecretSpecMaskinporten(t *testing.T) {
	client := maskinportenClientSetup("test-name")
	jwk, err := crypto.GenerateJwk()
	assert.NoError(t, err)

	spec, err := secrets.MaskinportenSpec(client, *jwk)
	assert.NoError(t, err, "should not error")

	stringData, err := secrets.MaskinportenStringData(*jwk, client)
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
			ObjectMeta: secrets.ObjectMeta(client, labels.MaskinportenLabels(client)),
			StringData: stringData,
			Type:       corev1.SecretTypeOpaque,
		}
		assert.NotEmpty(t, spec)
		assert.Equal(t, expected, spec)

		assert.Equal(t, corev1.SecretTypeOpaque, spec.Type, "Secret Type should be Opaque")
	})

	t.Run("StringData should contain expected fields and values", func(t *testing.T) {
		t.Run("Secret Data should contain "+secrets.MaskinportenJwkKey, func(t *testing.T) {
			expected, err := json.Marshal(jwk)
			assert.NoError(t, err)
			assert.Equal(t, string(expected), spec.StringData[secrets.MaskinportenJwkKey])
		})
		t.Run("Secret Data should contain "+secrets.MaskinportenWellKnownURL, func(t *testing.T) {
			expected := viper.GetString(config.DigDirMaskinportenBaseURL) + "/.well-known/oauth-authorization-server"
			assert.Equal(t, expected, spec.StringData[secrets.MaskinportenWellKnownURL])
		})
		t.Run("Secret Data should contain "+secrets.MaskinportenClientID, func(t *testing.T) {
			assert.Equal(t, client.Status.ClientID, spec.StringData[secrets.MaskinportenClientID])
		})
		t.Run("Secret Data should contain "+secrets.MaskinportenScopes+" with a single string of scopes separated by space", func(t *testing.T) {
			assert.Equal(t, secrets.ToScopesString(client.Spec.Scopes), spec.StringData[secrets.MaskinportenScopes])
		})
	})
}

func TestMaskinportenObjectMeta(t *testing.T) {
	app := maskinportenClientSetup("test-name")

	om := secrets.ObjectMeta(app, labels.MaskinportenLabels(app))

	t.Run("Name should be set", func(t *testing.T) {
		actual := om.GetName()
		assert.NotEmpty(t, actual)
		assert.Equal(t, app.Spec.SecretName, actual)
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
			labels.TypeLabelKey: labels.MaskinportenTypeLabelValue,
		}
		assert.NotEmpty(t, actualLabels, "Labels should not be empty")
		assert.Equal(t, expectedLabels, actualLabels, "Labels should be set")
	})
}

func maskinportenClientSetup(secretName string) *v1.MaskinportenClient {
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

func TestCreateSecretSpec(t *testing.T) {
	client := idportenClientSetup("test-name")
	jwk, err := crypto.GenerateJwk()
	assert.NoError(t, err)

	spec, err := secrets.IdportenSpec(client, *jwk)
	assert.NoError(t, err, "should not error")

	stringData, err := secrets.IdportenStringData(*jwk, client)
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
			ObjectMeta: secrets.ObjectMeta(client, labels.IDPortenLabels(client)),
			StringData: stringData,
			Type:       corev1.SecretTypeOpaque,
		}
		assert.NotEmpty(t, spec)
		assert.Equal(t, expected, spec)

		assert.Equal(t, corev1.SecretTypeOpaque, spec.Type, "Secret Type should be Opaque")
	})

	t.Run("StringData should contain expected fields and values", func(t *testing.T) {
		t.Run("Secret Data should contain "+secrets.IDPortenJwkKey, func(t *testing.T) {
			expected, err := json.Marshal(jwk)
			assert.NoError(t, err)
			assert.Equal(t, string(expected), spec.StringData[secrets.IDPortenJwkKey])
		})
		t.Run("Secret Data should contain "+secrets.IDPortenWellKnownURL, func(t *testing.T) {
			expected := viper.GetString(config.DigDirAuthBaseURL) + "/idporten-oidc-provider/.well-known/openid-configuration"
			assert.Equal(t, expected, spec.StringData[secrets.IDPortenWellKnownURL])
		})
		t.Run("Secret Data should contain "+secrets.IDPortenClientID, func(t *testing.T) {
			assert.Equal(t, client.Status.ClientID, spec.StringData[secrets.IDPortenClientID])
		})
		t.Run("Secret Data should contain "+secrets.IDPortenRedirectURI, func(t *testing.T) {
			assert.Equal(t, client.Spec.RedirectURI, spec.StringData[secrets.IDPortenRedirectURI])
		})
	})
}

func TestIdportenObjectMeta(t *testing.T) {
	app := idportenClientSetup("test-name")

	om := secrets.ObjectMeta(app, labels.IDPortenLabels(app))

	t.Run("Name should be set", func(t *testing.T) {
		actual := om.GetName()
		assert.NotEmpty(t, actual)
		assert.Equal(t, app.Spec.SecretName, actual)
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
			labels.TypeLabelKey: labels.IDPortenTypeLabelValue,
		}
		assert.NotEmpty(t, actualLabels, "Labels should not be empty")
		assert.Equal(t, expectedLabels, actualLabels, "Labels should be set")
	})
}

func idportenClientSetup(secretName string) *v1.IDPortenClient {
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
