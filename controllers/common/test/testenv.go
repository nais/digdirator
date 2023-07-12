package test

import (
	"context"
	"fmt"
	"io/ioutil"
	"net/http/httptest"
	"time"

	nais_io_v1 "github.com/nais/liberator/pkg/apis/nais.io/v1"
	"github.com/nais/liberator/pkg/crd"
	log "github.com/sirupsen/logrus"
	"gopkg.in/square/go-jose.v2"
	"k8s.io/client-go/kubernetes/scheme"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/envtest"
	"sigs.k8s.io/controller-runtime/pkg/log/zap"

	"github.com/nais/digdirator/controllers/common"
	"github.com/nais/digdirator/controllers/idportenclient"
	"github.com/nais/digdirator/controllers/maskinportenclient"
	"github.com/nais/digdirator/pkg/config"
)

const (
	Timeout              = time.Second * 5
	Interval             = time.Millisecond * 100
	ClientID             = "some-random-id"
	Scope                = "nav:arbeid/test/scope"
	ExposedConsumerOrgno = "111111111"
	UnusedSecret         = "unused-secret"
	AlreadyInUseSecret   = "in-use-by-pod"
)

func SetupTestEnv(clientID, scope, exposedConsumerOrgno string, handlerType HandlerType) (*envtest.Environment, *client.Client, error) {
	logger := zap.New(zap.UseDevMode(true))
	ctrl.SetLogger(logger)
	log.SetLevel(log.DebugLevel)

	crdPath := crd.YamlDirectory()
	testEnv := &envtest.Environment{
		CRDDirectoryPaths: []string{crdPath},
	}

	cfg, err := testEnv.Start()
	if err != nil {
		return nil, nil, err
	}

	err = nais_io_v1.AddToScheme(scheme.Scheme)
	if err != nil {
		return nil, nil, err
	}

	// +kubebuilder:scaffold:scheme

	mgr, err := ctrl.NewManager(cfg, ctrl.Options{
		Scheme:             scheme.Scheme,
		MetricsBindAddress: "0",
	})

	if err != nil {
		return nil, nil, err
	}

	cli := mgr.GetClient()

	digdiratorConfig, err := config.New()
	if err != nil {
		return nil, nil, fmt.Errorf("loading config: %v", err)
	}

	// TODO: consider replacing this with the KmsSigner or RsaSigner, however the KmsSigner requires a lot of mocking
	jwk, err := loadJwkFromPath("../common/testdata/testjwk")
	if err != nil {
		return nil, nil, fmt.Errorf("loading jwk credentials: %v", err)
	}

	signer, err := signerFromJwk(jwk)
	if err != nil {
		return nil, nil, fmt.Errorf("creating signer from jwk: %v", err)
	}

	testServer := httptest.NewServer(DigdirHandler(clientID, handlerType, scope, exposedConsumerOrgno))
	httpClient := testServer.Client()
	digdiratorConfig.ClusterName = "test-cluster"
	digdiratorConfig.DigDir.Admin.BaseURL = testServer.URL
	digdiratorConfig.DigDir.IDPorten.WellKnownURL = testServer.URL + "/.well-known/openid-configuration"
	digdiratorConfig.DigDir.Maskinporten.WellKnownURL = testServer.URL + "/.well-known/oauth-authorization-server"

	digdiratorConfig, err = digdiratorConfig.WithProviderMetadata(context.Background())
	if err != nil {
		return nil, nil, err
	}

	commonReconciler := common.NewReconciler(
		mgr.GetClient(),
		mgr.GetAPIReader(),
		mgr.GetScheme(),
		mgr.GetEventRecorderFor("digdirator"),
		digdiratorConfig,
		signer,
		httpClient,
		[]byte("client-id"),
	)

	switch handlerType {
	case IDPortenHandlerType:
		reconciler := idportenclient.NewReconciler(commonReconciler)
		err = reconciler.SetupWithManager(mgr)
		if err != nil {
			return nil, nil, err
		}
	case MaskinportenHandlerType:
		reconciler := maskinportenclient.NewReconciler(commonReconciler)
		err = reconciler.SetupWithManager(mgr)
		if err != nil {
			return nil, nil, err
		}
	}

	go func() {
		err = mgr.Start(ctrl.SetupSignalHandler())
		if err != nil {
			panic(err)
		}
	}()

	return testEnv, &cli, nil
}

func loadJwkFromPath(path string) (*jose.JSONWebKey, error) {
	creds, err := ioutil.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("loading JWK from path %s: %w", path, err)
	}

	jwk := &jose.JSONWebKey{}
	err = jwk.UnmarshalJSON(creds)
	if err != nil {
		return nil, fmt.Errorf("unmarshalling JWK: %w", err)
	}

	return jwk, nil
}

func signerFromJwk(jwk *jose.JSONWebKey) (jose.Signer, error) {
	signerOpts := jose.SignerOptions{}
	signerOpts.WithType("JWT")
	signerOpts.WithHeader("kid", jwk.KeyID)

	signer, err := jose.NewSigner(jose.SigningKey{Algorithm: jose.RS256, Key: jwk.Key}, &signerOpts)
	if err != nil {
		return nil, fmt.Errorf("creating jwt signer: %v", err)
	}
	return signer, nil
}
