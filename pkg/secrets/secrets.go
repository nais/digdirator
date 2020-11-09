package secrets

import (
	"github.com/nais/digdirator/controllers"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func Spec(data map[string]string, om metav1.ObjectMeta) *corev1.Secret {
	return &corev1.Secret{
		TypeMeta: metav1.TypeMeta{
			Kind:       "Secret",
			APIVersion: "v1",
		},
		ObjectMeta: om,
		StringData: data,
		Type:       corev1.SecretTypeOpaque,
	}
}

func ObjectMeta(instance controllers.Instance, labels map[string]string) metav1.ObjectMeta {
	return metav1.ObjectMeta{
		Name:      instance.SecretName(),
		Namespace: instance.NameSpace(),
		Labels:    labels,
	}
}
