package secrets_test

import (
	"github.com/nais/digdirator/pkg/crypto"
	"github.com/nais/digdirator/pkg/secrets"
	"testing"

	"github.com/nais/digdirator/pkg/labels"
	"github.com/stretchr/testify/assert"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func TestOpaqueSecret(t *testing.T) {
	client := idPortenClient("test-name")
	jwk, err := crypto.GenerateJwk()
	assert.NoError(t, err)

	stringData, err := secrets.IDPortenStringData(*jwk, client)
	assert.NoError(t, err, "should not error")

	expectedLabels := labels.IDPortenLabels(client)

	spec, err := secrets.OpaqueSecret(client, *jwk)
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
			ObjectMeta: metav1.ObjectMeta{
				Name:      client.SecretName(),
				Namespace: client.GetNamespace(),
				Labels:    expectedLabels,
			},
			StringData: stringData,
			Type:       corev1.SecretTypeOpaque,
		}
		assert.NotEmpty(t, spec)
		assert.Equal(t, expected, spec)
		assert.Equal(t, corev1.SecretTypeOpaque, spec.Type, "Secret Type should be Opaque")
	})
	t.Run("Name should be set", func(t *testing.T) {
		actual := spec.GetName()
		assert.NotEmpty(t, actual)
		assert.Equal(t, client.Spec.SecretName, actual)
	})

	t.Run("Namespace should be set", func(t *testing.T) {
		actual := spec.GetNamespace()
		assert.NotEmpty(t, actual)
		assert.Equal(t, client.GetNamespace(), actual)
	})
	t.Run("Labels should be set", func(t *testing.T) {
		actualLabels := spec.GetLabels()
		assert.NotEmpty(t, actualLabels, "Labels should not be empty")
		assert.Equal(t, expectedLabels, actualLabels, "Labels should be set")
	})
}
