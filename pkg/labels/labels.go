package labels

import (
	"github.com/nais/digdirator/controllers"
)

const (
	AppLabelKey                string = "app"
	TypeLabelKey               string = "type"
	IDPortenTypeLabelValue     string = "digdirator.nais.io"
	MaskinportenTypeLabelValue string = "maskinporten.digdirator.nais.io"
)

func MaskinportenLabels(instance controllers.Instance) map[string]string {
	return map[string]string{
		AppLabelKey:  instance.ClientName(),
		TypeLabelKey: MaskinportenTypeLabelValue,
	}
}

func IDPortenLabels(instance controllers.Instance) map[string]string {
	return map[string]string{
		AppLabelKey:  instance.ClientName(),
		TypeLabelKey: IDPortenTypeLabelValue,
	}
}
