package labels

import (
	"github.com/nais/digdirator/controllers"
)

const (
	AppLabelKey    string = "app"
	TypeLabelKey   string = "type"
	TypeLabelValue string = "digdirator.nais.io"
)

func DefaultLabels(instance controllers.Instance) map[string]string {
	return map[string]string{
		AppLabelKey:  instance.ClientName(),
		TypeLabelKey: TypeLabelValue,
	}
}
