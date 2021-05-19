package clients_test

import (
	"github.com/nais/digdirator/controllers/common"
	naisiov1 "github.com/nais/liberator/pkg/apis/nais.io/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func minimalIDPortenClient() *naisiov1.IDPortenClient {
	return &naisiov1.IDPortenClient{
		ObjectMeta: metav1.ObjectMeta{
			Name:        "test-app",
			Namespace:   "test-namespace",
			ClusterName: "test-cluster",
		},
		Spec: naisiov1.IDPortenClientSpec{
			ClientURI:   "",
			RedirectURI: "https://test.com",
			SecretName:  "test",
		},
		Status: naisiov1.DigdiratorStatus{
			SynchronizationHash:  "8b5ebee90b513411",
			SynchronizationState: common.EventSynchronized,
		},
	}
}

func minimalMaskinportenClient() *naisiov1.MaskinportenClient {
	return &naisiov1.MaskinportenClient{
		ObjectMeta: metav1.ObjectMeta{
			Name:        "test-app",
			Namespace:   "test-namespace",
			ClusterName: "test-cluster",
		},
		Spec: naisiov1.MaskinportenClientSpec{
			Scopes: naisiov1.MaskinportenScope{
				UsedScope: []naisiov1.UsedScope{
					{
						Name: "some-scope",
					},
				},
			},
		},
		Status: naisiov1.DigdiratorStatus{
			SynchronizationHash:  "94d732f6c1ff0b13",
			SynchronizationState: common.EventSynchronized,
		},
	}
}

func minimalMaskinportenWithScopeInternalExposedClient() *naisiov1.MaskinportenClient {
	return &naisiov1.MaskinportenClient{
		ObjectMeta: metav1.ObjectMeta{
			Name:        "test-app",
			Namespace:   "test-namespace",
			ClusterName: "test-cluster",
		},
		Spec: naisiov1.MaskinportenClientSpec{
			Scopes: naisiov1.MaskinportenScope{
				UsedScope: []naisiov1.UsedScope{
					{
						Name: "some-scope",
					},
				},
				ExposedScopes: []naisiov1.ExposedScope{
					{
						Name:                "my/scope",
						Enabled:             true,
						AtAgeMax:            30,
						Product:             "arbeid",
						AllowedIntegrations: []string{"maskinporten"},
						Consumers: []naisiov1.ExposedScopeConsumer{
							{
								Orgno: "1010101010",
							},
						},
					},
				},
			},
		},
		Status: naisiov1.DigdiratorStatus{
			SynchronizationHash:  "76cd5f86dc47f6cf",
			SynchronizationState: common.EventSynchronized,
		},
	}
}
