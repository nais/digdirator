package clients

import (
	nais_io_v1 "github.com/nais/liberator/pkg/apis/nais.io/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

const (
	AppLabelKey                string = "app"
	TypeLabelKey               string = "type"
	IDPortenTypeLabelValue     string = "digdirator.nais.io"
	MaskinportenTypeLabelValue string = "maskinporten.digdirator.nais.io"
)

func MakeLabels(instance Instance) map[string]string {
	switch instance.(type) {
	case *nais_io_v1.IDPortenClient:
		return idPortenLabels(instance.(*nais_io_v1.IDPortenClient))
	case *nais_io_v1.MaskinportenClient:
		return maskinportenLabels(instance.(*nais_io_v1.MaskinportenClient))
	}
	return nil
}

func maskinportenLabels(instance metav1.Object) map[string]string {
	return map[string]string{
		AppLabelKey:  instance.GetName(),
		TypeLabelKey: MaskinportenTypeLabelValue,
	}
}

func idPortenLabels(instance metav1.Object) map[string]string {
	return map[string]string{
		AppLabelKey:  instance.GetName(),
		TypeLabelKey: IDPortenTypeLabelValue,
	}
}
