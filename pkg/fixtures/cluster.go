package fixtures

import (
	"context"
	"fmt"
	v1 "github.com/nais/digdirator/api/v1"
	"github.com/nais/digdirator/pkg/labels"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"time"

	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

type ClusterFixtures struct {
	client.Client
	Config
	idPortenClient     *v1.IDPortenClient
	maskinportenClient *v1.MaskinportenClient
	namespace          *corev1.Namespace
	pod                *corev1.Pod
	podEnvFrom         *corev1.Pod
	unusedSecret       *corev1.Secret
}

type Config struct {
	DigdirClientName string
	NamespaceName    string
	SecretName       string
	UnusedSecretName string
}

type resource struct {
	client.ObjectKey
	runtime.Object
}

func New(cli client.Client, config Config) ClusterFixtures {
	return ClusterFixtures{Client: cli, Config: config}
}

func (c ClusterFixtures) MinimalConfig() ClusterFixtures {
	return c.WithNamespace()
}

func (c ClusterFixtures) WithNamespace() ClusterFixtures {
	c.namespace = &corev1.Namespace{
		TypeMeta: metav1.TypeMeta{
			Kind:       "Namespace",
			APIVersion: "v1",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name: c.NamespaceName,
		},
	}
	return c
}

func (c ClusterFixtures) WithIDPortenClient() ClusterFixtures {
	key := types.NamespacedName{
		Namespace: c.NamespaceName,
		Name:      c.DigdirClientName,
	}

	spec := v1.IDPortenClientSpec{
		ClientURI:              "clienturi",
		RedirectURI:            "https://my-app.nais.io",
		SecretName:             c.SecretName,
		FrontchannelLogoutURI:  "frontChannelLogoutURI",
		PostLogoutRedirectURIs: []string{"postLogoutRedirectURI"},
		RefreshTokenLifetime:   0,
	}
	c.idPortenClient = &v1.IDPortenClient{
		ObjectMeta: metav1.ObjectMeta{
			Name:        key.Name,
			Namespace:   key.Namespace,
			ClusterName: "test-cluster",
		},
		Spec: spec,
	}
	return c
}

func (c ClusterFixtures) WithMaskinportenClient() ClusterFixtures {
	key := types.NamespacedName{
		Namespace: c.NamespaceName,
		Name:      c.DigdirClientName,
	}

	spec := v1.MaskinportenClientSpec{
		SecretName: c.SecretName,
		Scopes: []v1.MaskinportenScope{
			{Scope: "scopes"},
		},
	}
	c.maskinportenClient = &v1.MaskinportenClient{
		ObjectMeta: metav1.ObjectMeta{
			Name:        key.Name,
			Namespace:   key.Namespace,
			ClusterName: "test-cluster",
		},
		Spec: spec,
	}
	return c
}

func (c ClusterFixtures) WithPods() ClusterFixtures {
	key := types.NamespacedName{
		Namespace: c.NamespaceName,
		Name:      c.DigdirClientName,
	}
	c.pod = &corev1.Pod{
		TypeMeta: metav1.TypeMeta{
			Kind:       "Pod",
			APIVersion: "v1",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      key.Name,
			Namespace: key.Namespace,
			Labels: map[string]string{
				labels.AppLabelKey: key.Name,
			},
		},
		Spec: corev1.PodSpec{
			Containers: []corev1.Container{
				{
					Name:  "main",
					Image: "foo",
				},
			},
			Volumes: []corev1.Volume{
				{
					Name: "foo",
					VolumeSource: corev1.VolumeSource{
						Secret: &corev1.SecretVolumeSource{
							SecretName: c.SecretName,
						},
					},
				},
			},
		},
	}
	c.podEnvFrom = &corev1.Pod{
		TypeMeta: metav1.TypeMeta{
			Kind:       "Pod",
			APIVersion: "v1",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      fmt.Sprintf("%s-envfrom", key.Name),
			Namespace: c.NamespaceName,
			Labels: map[string]string{
				labels.AppLabelKey: key.Name,
			},
		},
		Spec: corev1.PodSpec{
			Containers: []corev1.Container{
				{
					Name:  "main",
					Image: "foo",
					EnvFrom: []corev1.EnvFromSource{
						{
							SecretRef: &corev1.SecretEnvSource{
								LocalObjectReference: corev1.LocalObjectReference{
									Name: c.SecretName,
								},
							},
						},
					},
				},
			},
		},
	}
	return c
}

func (c ClusterFixtures) WithUnusedSecret(label string) ClusterFixtures {
	c.unusedSecret = &corev1.Secret{
		TypeMeta: metav1.TypeMeta{
			Kind:       "Secret",
			APIVersion: "v1",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      c.UnusedSecretName,
			Namespace: c.NamespaceName,
			Labels: map[string]string{
				labels.AppLabelKey:  c.DigdirClientName,
				labels.TypeLabelKey: label,
			},
		},
	}
	return c
}

func (c ClusterFixtures) Setup() error {
	ctx := context.Background()
	if c.namespace != nil {
		if err := c.Create(ctx, c.namespace); err != nil {
			return err
		}
	}
	if c.idPortenClient != nil {
		if err := c.Create(ctx, c.idPortenClient); err != nil {
			return err
		}
	}
	if c.maskinportenClient != nil {
		if err := c.Create(ctx, c.maskinportenClient); err != nil {
			return err
		}
	}
	if c.pod != nil {
		if err := c.Create(ctx, c.pod); err != nil {
			return err
		}
	}
	if c.podEnvFrom != nil {
		if err := c.Create(ctx, c.podEnvFrom); err != nil {
			return err
		}
	}
	if c.unusedSecret != nil {
		if err := c.Create(ctx, c.unusedSecret); err != nil {
			return err
		}
	}
	return c.waitForClusterResources(ctx)
}

func (c ClusterFixtures) waitForClusterResources(ctx context.Context) error {
	resources := make([]resource, 0)
	if c.unusedSecret != nil {
		resources = append(resources, resource{
			ObjectKey: client.ObjectKey{
				Namespace: c.NamespaceName,
				Name:      c.UnusedSecretName,
			},
			Object: &corev1.Secret{},
		})
	}
	if c.pod != nil {
		resources = append(resources, resource{
			ObjectKey: client.ObjectKey{
				Namespace: c.NamespaceName,
				Name:      c.DigdirClientName,
			},
			Object: &corev1.Pod{},
		})
	}
	if c.podEnvFrom != nil {
		resources = append(resources, resource{
			ObjectKey: client.ObjectKey{
				Namespace: c.NamespaceName,
				Name:      fmt.Sprintf("%s-envfrom", c.DigdirClientName),
			},
			Object: &corev1.Pod{},
		})
	}
	if c.idPortenClient != nil {
		resources = append(resources, resource{
			ObjectKey: client.ObjectKey{
				Namespace: c.NamespaceName,
				Name:      c.DigdirClientName,
			},
			Object: &v1.IDPortenClient{},
		})
	}
	if c.maskinportenClient != nil {
		resources = append(resources, resource{
			ObjectKey: client.ObjectKey{
				Namespace: c.NamespaceName,
				Name:      c.DigdirClientName,
			},
			Object: &v1.MaskinportenClient{},
		})
	}

	timeout := time.NewTimer(5 * time.Second)
	ticker := time.NewTicker(100 * time.Millisecond)

	for {
		select {
		case <-timeout.C:
			return fmt.Errorf("timeout while waiting for cluster fixtures setup synchronization")
		case <-ticker.C:
			exists, err := allExists(ctx, c.Client, resources)
			if err != nil {
				return err
			}
			if exists {
				return nil
			}
		}
	}
}

func allExists(ctx context.Context, cli client.Client, resources []resource) (bool, error) {
	for _, resource := range resources {
		err := cli.Get(ctx, resource.ObjectKey, resource.Object)
		if err == nil {
			continue
		}
		if errors.IsNotFound(err) {
			return false, nil
		} else {
			return false, err
		}
	}
	return true, nil
}
