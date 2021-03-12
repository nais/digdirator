package annotations

import (
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/nais/digdirator/pkg/clients"
)

const (
	SkipKey   = "digdirator.nais.io/skip"
	SkipValue = "true"
)

func Set(resource v1.ObjectMetaAccessor, key, value string) {
	a := resource.GetObjectMeta().GetAnnotations()
	if a == nil {
		a = make(map[string]string)
	}
	a[key] = value
	resource.GetObjectMeta().SetAnnotations(a)
}

func HasAnnotation(resource v1.ObjectMetaAccessor, key string) (string, bool) {
	value, found := resource.GetObjectMeta().GetAnnotations()[key]
	return value, found
}

func HasSkipFlag(instance clients.Instance) bool {
	_, found := HasAnnotation(instance, SkipKey)
	return found
}
