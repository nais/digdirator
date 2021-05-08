package clients_test

import (
	"github.com/nais/digdirator/controllers/common"
	nais_io_v1 "github.com/nais/liberator/pkg/apis/nais.io/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func minimalIDPortenClient() *nais_io_v1.IDPortenClient {
	return &nais_io_v1.IDPortenClient{
		ObjectMeta: metav1.ObjectMeta{
			Name:        "test-app",
			Namespace:   "test-namespace",
			ClusterName: "test-cluster",
		},
		Spec: nais_io_v1.IDPortenClientSpec{
			ClientURI:   "",
			RedirectURI: "https://test.com",
			SecretName:  "test",
		},
		Status: nais_io_v1.DigdiratorStatus{
			SynchronizationHash:  "8b5ebee90b513411",
			SynchronizationState: common.EventSynchronized,
		},
	}
}

func minimalMaskinportenClient() *nais_io_v1.MaskinportenClient {
	return &nais_io_v1.MaskinportenClient{
		ObjectMeta: metav1.ObjectMeta{
			Name:        "test-app",
			Namespace:   "test-namespace",
			ClusterName: "test-cluster",
		},
		Spec: nais_io_v1.MaskinportenClientSpec{
			Scopes: []nais_io_v1.MaskinportenScope{
				{
					Name: "some-scope",
				},
			},
		},
		Status: nais_io_v1.DigdiratorStatus{
			SynchronizationHash:  "4a4e08c51548e46e",
			SynchronizationState: common.EventSynchronized,
		},
	}
}

func minimalMaskinportenWithScopeInternalExternalClient() *nais_io_v1.MaskinportenClient {
	return &nais_io_v1.MaskinportenClient{
		ObjectMeta: metav1.ObjectMeta{
			Name:        "test-app",
			Namespace:   "test-namespace",
			ClusterName: "test-cluster",
		},
		Spec: nais_io_v1.MaskinportenClientSpec{
			Scope: nais_io_v1.MaskinportenScopeSpec{
				Internal: []nais_io_v1.InternalScope{
					{
						Name: "some-scope",
					},
				},
				External: []nais_io_v1.ExternalScope{
					{
						Name:                "my-scope",
						AtAgeMax:            30,
						AllowedIntegrations: []string{"maskinporten"},
						Consumers: []nais_io_v1.ExternalScopeConsumer{
							{
								Orgno: "1010101010",
							},
						},
					},
				},
			},
		},
		Status: nais_io_v1.DigdiratorStatus{
			SynchronizationHash:  "d3ad1ee0de188c6f",
			SynchronizationState: common.EventSynchronized,
		},
	}
}
