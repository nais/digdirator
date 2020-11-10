package common

import (
	"github.com/nais/digdirator/pkg/digdir/types"
	"github.com/nais/digdirator/pkg/util"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
)

type Instance interface {
	metav1.Object
	runtime.Object
	StatusClientID() string
	Description() string
	SecretName() string
	Labels() map[string]string
	IntegrationType() types.IntegrationType
}

func InstanceIsBeingDeleted(instance Instance) bool {
	return !instance.GetDeletionTimestamp().IsZero()
}

func HasFinalizer(instance Instance, finalizerName string) bool {
	return util.ContainsString(instance.GetFinalizers(), finalizerName)
}
