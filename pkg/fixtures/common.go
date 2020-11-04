package fixtures

import (
	"context"
	"k8s.io/apimachinery/pkg/runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

type ClusterFixtures interface {
	WithPod()
	WithUnusedSecret()
	AllExists(context.Context, client.Client, []Resource) (bool, error)
}

type Config struct {
	DigidirClientName string
	NamespaceName     string
	SecretName        string
	UnusedSecretName  string
}

type Resource struct {
	client.ObjectKey
	runtime.Object
}
