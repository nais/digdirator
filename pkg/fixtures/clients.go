package fixtures

import (
	"github.com/nais/digdirator/controllers/common"
	naisiov1 "github.com/nais/liberator/pkg/apis/nais.io/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/utils/ptr"
)

func MinimalIDPortenClient() *naisiov1.IDPortenClient {
	return &naisiov1.IDPortenClient{
		ObjectMeta: metav1.ObjectMeta{
			Name:       "test-app",
			Namespace:  "test-namespace",
			Generation: 1,
		},
		Spec: naisiov1.IDPortenClientSpec{
			ClientURI: "",
			RedirectURIs: []naisiov1.IDPortenURI{
				"https://test.com",
			},
			SecretName: "test",
		},
		Status: naisiov1.DigdiratorStatus{
			SynchronizationHash:  "de6ecbc3b6cb148b",
			SynchronizationState: common.EventSynchronized,
			ClientID:             "test-idporten",
			ObservedGeneration:   ptr.To(int64(1)),
		},
	}
}

func MinimalMaskinportenClient() *naisiov1.MaskinportenClient {
	return &naisiov1.MaskinportenClient{
		ObjectMeta: metav1.ObjectMeta{
			Name:       "test-app",
			Namespace:  "test-namespace",
			Generation: 1,
		},
		Spec: naisiov1.MaskinportenClientSpec{
			Scopes: naisiov1.MaskinportenScope{
				ConsumedScopes: []naisiov1.ConsumedScope{
					{
						Name: "some-scope",
					},
				},
			},
		},
		Status: naisiov1.DigdiratorStatus{
			SynchronizationHash:  "9829660b73e52236",
			SynchronizationState: common.EventSynchronized,
			ClientID:             "test-maskinporten",
			ObservedGeneration:   ptr.To(int64(1)),
		},
	}
}

func MinimalMaskinportenWithScopeInternalExposedClient() *naisiov1.MaskinportenClient {
	atMaxAge := 30
	return &naisiov1.MaskinportenClient{
		ObjectMeta: metav1.ObjectMeta{
			Name:       "test-app",
			Namespace:  "test-namespace",
			Generation: 1,
		},
		Spec: naisiov1.MaskinportenClientSpec{
			Scopes: naisiov1.MaskinportenScope{
				ConsumedScopes: []naisiov1.ConsumedScope{
					{
						Name: "some-scope",
					},
				},
				ExposedScopes: []naisiov1.ExposedScope{
					{
						Name:                "my/scope",
						Enabled:             true,
						AtMaxAge:            &atMaxAge,
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
			SynchronizationHash:  "18a7e807ed742be3",
			SynchronizationState: common.EventSynchronized,
			ClientID:             "test-maskinporten",
			ObservedGeneration:   ptr.To(int64(1)),
		},
	}
}
