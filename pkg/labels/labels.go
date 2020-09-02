package labels

import (
	v1 "github.com/nais/digdirator/api/v1"
)

const (
	AppLabelKey    string = "app"
	TypeLabelKey   string = "type"
	TypeLabelValue string = "digdirator.nais.io"
)

func DefaultLabels(instance *v1.IDPortenClient) map[string]string {
	return map[string]string{
		AppLabelKey:  instance.GetName(),
		TypeLabelKey: TypeLabelValue,
	}
}
