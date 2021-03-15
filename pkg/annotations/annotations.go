package annotations

import (
	nais_io_v1 "github.com/nais/liberator/pkg/apis/nais.io/v1"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/nais/digdirator/pkg/clients"
)

const (
	SkipKey   = "digdirator.nais.io/skip"
	SkipValue = "true"
)

func HasAnnotation(resource v1.ObjectMetaAccessor, key string) (string, bool) {
	value, found := resource.GetObjectMeta().GetAnnotations()[key]
	return value, found
}

func HasSkipFlag(instance clients.Instance) bool {
	switch instance.(type) {
	case *nais_io_v1.IDPortenClient:
		return hasSkipFlag(instance.(*nais_io_v1.IDPortenClient))
	case *nais_io_v1.MaskinportenClient:
		return hasSkipFlag(instance.(*nais_io_v1.MaskinportenClient))
	default:
		return false
	}
}

func hasSkipFlag(resource v1.ObjectMetaAccessor) bool {
	_, found := HasAnnotation(resource, SkipKey)
	return found
}

func Set(instance clients.Instance, key, value string) {
	switch instance.(type) {
	case *nais_io_v1.IDPortenClient:
		set(instance.(*nais_io_v1.IDPortenClient), key, value)
	case *nais_io_v1.MaskinportenClient:
		set(instance.(*nais_io_v1.MaskinportenClient), key, value)
	}
}

func set(instance v1.ObjectMetaAccessor, key, value string) {
	a := instance.GetObjectMeta().GetAnnotations()
	if a == nil {
		a = make(map[string]string)
	}
	a[key] = value
	instance.GetObjectMeta().SetAnnotations(a)
}
