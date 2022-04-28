package annotations

import (
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

const (
	DeleteKey = "digdir.nais.io/delete"
)

func HasAnnotation(resource v1.ObjectMetaAccessor, key string) (string, bool) {
	value, found := resource.GetObjectMeta().GetAnnotations()[key]
	return value, found
}

func HasDeleteAnnotation(resource v1.ObjectMetaAccessor) bool {
	_, found := HasAnnotation(resource, DeleteKey)
	return found
}
