package labels

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

const (
	AppLabelKey                string = "app"
	TypeLabelKey               string = "type"
	IDPortenTypeLabelValue     string = "digdirator.nais.io"
	MaskinportenTypeLabelValue string = "maskinporten.digdirator.nais.io"
)

func MaskinportenLabels(instance metav1.Object) map[string]string {
	return map[string]string{
		AppLabelKey:  instance.GetName(),
		TypeLabelKey: MaskinportenTypeLabelValue,
	}
}

func IDPortenLabels(instance metav1.Object) map[string]string {
	return map[string]string{
		AppLabelKey:  instance.GetName(),
		TypeLabelKey: IDPortenTypeLabelValue,
	}
}
