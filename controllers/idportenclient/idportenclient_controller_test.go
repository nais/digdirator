package idportenclient

import (
	"context"
	"fmt"
	v1 "github.com/nais/digdirator/api/v1"
	"github.com/nais/digdirator/pkg/config"
	"github.com/nais/digdirator/pkg/crypto"
	"github.com/nais/digdirator/pkg/fixtures"
	"github.com/nais/digdirator/pkg/idporten"
	log "github.com/sirupsen/logrus"
	"github.com/stretchr/testify/assert"
	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/kubernetes/scheme"
	"os"
	"path/filepath"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/envtest"
	"sigs.k8s.io/controller-runtime/pkg/log/zap"
	"testing"
	"time"
	// +kubebuilder:scaffold:imports
)

const (
	timeout  = time.Second * 5
	interval = time.Millisecond * 100
)

var cli client.Client

func TestMain(m *testing.M) {
	testEnv, err := setup()
	if err != nil {
		os.Exit(1)
	}
	code := m.Run()
	_ = testEnv.Stop()
	os.Exit(code)
}

func setup() (*envtest.Environment, error) {
	logger := zap.New(zap.UseDevMode(true))
	ctrl.SetLogger(logger)
	log.SetLevel(log.DebugLevel)

	testEnv := &envtest.Environment{
		CRDDirectoryPaths: []string{filepath.Join("..", "..", "config", "crd")},
	}

	cfg, err := testEnv.Start()
	if err != nil {
		return nil, err
	}

	err = v1.AddToScheme(scheme.Scheme)
	if err != nil {
		return nil, err
	}

	// +kubebuilder:scaffold:scheme

	mgr, err := ctrl.NewManager(cfg, ctrl.Options{
		Scheme: scheme.Scheme,
	})

	if err != nil {
		return nil, err
	}

	cli = mgr.GetClient()

	digdiratorConfig, err := config.New()

	jwk, err := crypto.LoadJwkFromPath("testdata/testjwk")
	if err != nil {
		return nil, fmt.Errorf("loading jwk credentials: %v", err)
	}

	signer, err := crypto.SignerFromJwk(jwk)
	if err != nil {
		return nil, fmt.Errorf("creating signer from jwk: %v", err)
	}

	err = (&Reconciler{
		Client:         cli,
		Reader:         mgr.GetAPIReader(),
		Scheme:         mgr.GetScheme(),
		IDPortenClient: idporten.NewClient(signer, *digdiratorConfig),
		Recorder:       mgr.GetEventRecorderFor("digdirator"),
		Config:         digdiratorConfig,
	}).SetupWithManager(mgr)

	if err != nil {
		return nil, err
	}

	go func() {
		err = mgr.Start(ctrl.SetupSignalHandler())
		if err != nil {
			panic(err)
		}
	}()

	return testEnv, nil
}

func TestIDPortenController(t *testing.T) {
	// set up preconditions for cluster
	clusterFixtures := fixtures.New(cli, fixtures.Config{
		IDPortenClientName: "test-client",
		NamespaceName:      "test-namespace",
	}).MinimalConfig()

	if err := clusterFixtures.Setup(); err != nil {
		t.Fatalf("failed to set up cluster fixtures: %v", err)
	}

	instance := &v1.IDPortenClient{}
	key := client.ObjectKey{
		Name:      "test-client",
		Namespace: "test-namespace",
	}
	assert.Eventually(t, resourceExists(key, instance), timeout, interval, "IDPortenClient should exist")
}

func resourceExists(key client.ObjectKey, instance runtime.Object) func() bool {
	return func() bool {
		err := cli.Get(context.Background(), key, instance)
		return !errors.IsNotFound(err)
	}
}
