package test

import (
	"encoding/base64"
	"fmt"
	v1 "github.com/nais/digdirator/api/v1"
	"github.com/nais/digdirator/controllers/common"
	"github.com/nais/digdirator/controllers/idportenclient"
	"github.com/nais/digdirator/controllers/maskinportenclient"
	"github.com/nais/digdirator/pkg/config"
	log "github.com/sirupsen/logrus"
	"gopkg.in/square/go-jose.v2"
	"io/ioutil"
	"k8s.io/client-go/kubernetes/scheme"
	"net/http/httptest"
	"path/filepath"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/envtest"
	"sigs.k8s.io/controller-runtime/pkg/log/zap"
	"time"
)

const (
	Timeout  = time.Second * 5
	Interval = time.Millisecond * 100
)

func SetupTestEnv(clientID string, handlerType HandlerType) (*envtest.Environment, *client.Client, error) {
	logger := zap.New(zap.UseDevMode(true))
	ctrl.SetLogger(logger)
	log.SetLevel(log.DebugLevel)

	testEnv := &envtest.Environment{
		CRDDirectoryPaths: []string{filepath.Join("..", "..", "config", "crd")},
	}

	cfg, err := testEnv.Start()
	if err != nil {
		return nil, nil, err
	}

	err = v1.AddToScheme(scheme.Scheme)
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

	testServer := httptest.NewServer(DigdirHandler(clientID, handlerType))
	httpClient := testServer.Client()
	digdiratorConfig.ClusterName = "test-cluster"
	digdiratorConfig.DigDir.Auth.BaseURL = testServer.URL
	digdiratorConfig.DigDir.IDPorten.BaseURL = testServer.URL

	commonReconciler := common.NewReconciler(
		mgr.GetClient(),
		mgr.GetAPIReader(),
		mgr.GetScheme(),
		mgr.GetEventRecorderFor("digdirator"),
		digdiratorConfig,
		signer,
		httpClient,
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
	signerOpts.WithHeader("x5c", extractX5c(jwk))

	signer, err := jose.NewSigner(jose.SigningKey{Algorithm: jose.RS256, Key: jwk.Key}, &signerOpts)
	if err != nil {
		return nil, fmt.Errorf("creating jwt signer: %v", err)
	}
	return signer, nil
}

func extractX5c(jwk *jose.JSONWebKey) []string {
	x5c := make([]string, 0)
	for _, cert := range jwk.Certificates {
		x5c = append(x5c, base64.StdEncoding.EncodeToString(cert.Raw))
	}
	return x5c
}
