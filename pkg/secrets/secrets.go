package secrets

import (
	v1 "github.com/nais/digdirator/api/v1"
	"gopkg.in/square/go-jose.v2"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func OpaqueSecret(instance v1.Instance, jwk jose.JSONWebKey) (*corev1.Secret, error) {
	stringData, err := instance.CreateSecretData(jwk)
	if err != nil {
		return nil, err
	}

	return &corev1.Secret{
		TypeMeta: metav1.TypeMeta{
			Kind:       "Secret",
			APIVersion: "v1",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      instance.GetSecretName(),
			Namespace: instance.GetNamespace(),
			Labels:    instance.MakeLabels(),
		},
		StringData: stringData,
		Type:       corev1.SecretTypeOpaque,
	}, nil
}
