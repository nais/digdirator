package annotations

import (
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

const (
	SkipKey   = "digdir.nais.io/skip"
	DeleteKey = "digdir.nais.io/delete"
)

func HasAnnotation(resource v1.ObjectMetaAccessor, key string) (string, bool) {
	value, found := resource.GetObjectMeta().GetAnnotations()[key]
	return value, found
}

func HasSkipAnnotation(resource v1.ObjectMetaAccessor) bool {
	_, found := HasAnnotation(resource, SkipKey)
	return found
}

func HasDeleteAnnotation(resource v1.ObjectMetaAccessor) bool {
	_, found := HasAnnotation(resource, DeleteKey)
	return found
}

func Set(instance v1.ObjectMetaAccessor, key, value string) {
	a := instance.GetObjectMeta().GetAnnotations()
	if a == nil {
		a = make(map[string]string)
	}
	a[key] = value
	instance.GetObjectMeta().SetAnnotations(a)
}
