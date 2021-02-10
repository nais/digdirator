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
			SynchronizationHash:  "c8cc08293399879",
			SynchronizationState: common.EventSynchronized,
		},
	}
}
