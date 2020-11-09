package secrets

import (
	"fmt"
	v1 "github.com/nais/digdirator/api/v1"
	"github.com/nais/digdirator/controllers/common"
	"gopkg.in/square/go-jose.v2"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func OpaqueSecret(instance common.Instance, jwk jose.JSONWebKey) (*corev1.Secret, error) {
	var stringData map[string]string
	var err error

	switch instance.(type) {
	case *v1.IDPortenClient:
		client := instance.(*v1.IDPortenClient)
		stringData, err = IDPortenStringData(jwk, client)
	case *v1.MaskinportenClient:
		client := instance.(*v1.MaskinportenClient)
		stringData, err = MaskinportenStringData(jwk, client)
	default:
		return nil, fmt.Errorf("instance does not implement 'common.Instance'")
	}

	if err != nil {
		return nil, err
	}

	return &corev1.Secret{
		TypeMeta: metav1.TypeMeta{
			Kind:       "Secret",
			APIVersion: "v1",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      instance.SecretName(),
			Namespace: instance.GetNamespace(),
			Labels:    instance.Labels(),
		},
		StringData: stringData,
		Type:       corev1.SecretTypeOpaque,
	}, nil
}
