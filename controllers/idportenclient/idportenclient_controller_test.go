package idportenclient

import (
	"context"
	"encoding/base64"
	"fmt"
	v1 "github.com/nais/digdirator/api/v1"
	"github.com/nais/digdirator/pkg/config"
	"github.com/nais/digdirator/pkg/fixtures"
	"github.com/nais/digdirator/pkg/labels"
	"github.com/nais/digdirator/pkg/secrets"
	log "github.com/sirupsen/logrus"
	"github.com/stretchr/testify/assert"
	"gopkg.in/square/go-jose.v2"
	"io/ioutil"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/kubernetes/scheme"
	"net/http"
	"net/http/httptest"
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
	clientId = "some-random-id"
)

var cli client.Client
var clientExists bool

func TestMain(m *testing.M) {
	testEnv, err := setup()
	if err != nil {
		os.Exit(1)
	}
	code := m.Run()
	_ = testEnv.Stop()
	os.Exit(code)
}

func idportenHandler() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		switch {
		case r.URL.Path == "/idporten-oidc-provider/token":
			response := `{ "access_token": "token" }`
			_, _ = w.Write([]byte(response))
		// GET (list) clients
		case r.URL.Path == "/clients" && r.Method == http.MethodGet:
			var path string
			if clientExists {
				path = "testdata/list-response-exists.json"
			} else {
				path = "testdata/list-response.json"
			}
			response, _ := ioutil.ReadFile(path)
			_, _ = w.Write(response)
		// POST (create) client
		case r.URL.Path == "/clients" && r.Method == http.MethodPost:
			response, _ := ioutil.ReadFile("testdata/create-response.json")
			_, _ = w.Write(response)
		// PUT (update) existing client
		case r.URL.Path == fmt.Sprintf("/clients/%s", clientId) && r.Method == http.MethodPut:
			response, _ := ioutil.ReadFile("testdata/update-response.json")
			_, _ = w.Write(response)
		// DELETE existing client
		case r.URL.Path == fmt.Sprintf("/clients/%s", clientId) && r.Method == http.MethodDelete:
			w.WriteHeader(http.StatusOK)
		// POST JWKS (overwriting)
		case r.URL.Path == fmt.Sprintf("/clients/%s/jwks", clientId) && r.Method == http.MethodPost:
			var path string
			if clientExists {
				path = "testdata/register-jwks-response-exists.json"
			} else {
				path = "testdata/register-jwks-response.json"
			}
			response, _ := ioutil.ReadFile(path)
			_, _ = w.Write(response)
		}
	}
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
		Scheme:             scheme.Scheme,
		MetricsBindAddress: "0",
	})

	if err != nil {
		return nil, err
	}

	cli = mgr.GetClient()

	digdiratorConfig, err := config.New()
	if err != nil {
		return nil, fmt.Errorf("loading config: %v", err)
	}

	// TODO: consider replacing this with the KmsSigner or RsaSigner, however the KmsSigner requires a lot of mocking
	jwk, err := loadJwkFromPath("testdata/testjwk")
	if err != nil {
		return nil, fmt.Errorf("loading jwk credentials: %v", err)
	}

	signer, err := signerFromJwk(jwk)
	if err != nil {
		return nil, fmt.Errorf("creating signer from jwk: %v", err)
	}

	testServer := httptest.NewServer(idportenHandler())
	httpClient := testServer.Client()
	digdiratorConfig.ClusterName = "test-cluster"
	digdiratorConfig.DigDir.Auth.BaseURL = testServer.URL
	digdiratorConfig.DigDir.IDPorten.BaseURL = testServer.URL

	err = (&Reconciler{
		Client:     cli,
		Reader:     mgr.GetAPIReader(),
		Scheme:     mgr.GetScheme(),
		Recorder:   mgr.GetEventRecorderFor("digdirator"),
		Config:     digdiratorConfig,
		Signer:     signer,
		HttpClient: httpClient,
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
	cfg := fixtures.Config{
		DigidirClientName: "test-client",
		NamespaceName:     "test-namespace",
		SecretName:        "test-secret",
		UnusedSecretName:  "test-unused-secret",
	}

	// set up preconditions for cluster
	clusterFixtures := fixtures.New(cli, cfg).MinimalConfig().WithIDPortenClient().WithPod().WithUnusedSecret(labels.IDPortenTypeLabelValue)

	// create IDPortenClient
	if err := clusterFixtures.Setup(); err != nil {
		t.Fatalf("failed to set up cluster fixtures: %v", err)
	}

	instance := &v1.IDPortenClient{}
	key := client.ObjectKey{
		Name:      "test-client",
		Namespace: "test-namespace",
	}
	assert.Eventually(t, resourceExists(key, instance), timeout, interval, "IDPortenClient should exist")
	assert.Eventually(t, func() bool {
		err := cli.Get(context.Background(), key, instance)
		assert.NoError(t, err)
		b, err := instance.HashUnchanged()
		assert.NoError(t, err)
		return b
	}, timeout, interval, "IDPortenClient should be synchronized")
	assert.NotEmpty(t, instance.Status.ClientID)
	assert.NotEmpty(t, instance.Status.KeyIDs)
	assert.NotEmpty(t, instance.Status.ProvisionHash)
	assert.NotEmpty(t, instance.Status.CorrelationID)
	assert.NotEmpty(t, instance.Status.Timestamp)

	assert.Equal(t, clientId, instance.Status.ClientID)
	assert.Contains(t, instance.Status.KeyIDs, "some-keyid")
	assert.Len(t, instance.Status.KeyIDs, 1)

	assertSecretExists(t, cfg.SecretName, cfg.NamespaceName, instance)

	assert.Eventually(t, resourceDoesNotExist(client.ObjectKey{
		Namespace: cfg.NamespaceName,
		Name:      cfg.UnusedSecretName,
	}, &corev1.Secret{}), timeout, interval, "unused Secret should not exist")
	clientExists = true

	// update IDPortenClient
	previousSecretName := cfg.SecretName
	previousHash := instance.Status.ProvisionHash

	// set new secretname in spec -> trigger update
	instance.Spec.SecretName = "new-secret-name"
	err := cli.Update(context.Background(), instance)
	assert.NoError(t, err)

	// new hash should be set in status
	assert.Eventually(t, func() bool {
		err := cli.Get(context.Background(), key, instance)
		assert.NoError(t, err)
		return previousHash != instance.Status.ProvisionHash
	}, timeout, interval, "new hash should be set")

	assert.Equal(t, clientId, instance.Status.ClientID, "client ID should still match")
	assert.Len(t, instance.Status.KeyIDs, 2, "should contain two key IDs")
	assert.Contains(t, instance.Status.KeyIDs, "some-keyid", "previous key should still be valid")
	assert.Contains(t, instance.Status.KeyIDs, "some-new-keyid", "new key should be valid")

	// new secret should exist
	assertSecretExists(t, instance.Spec.SecretName, cfg.NamespaceName, instance)

	// old secret should still exist
	assertSecretExists(t, previousSecretName, cfg.NamespaceName, instance)

	// delete IDPortenClient
	err = cli.Delete(context.Background(), instance)

	assert.NoError(t, err, "deleting IDPortenClient")

	assert.Eventually(t, resourceDoesNotExist(key, instance), timeout, interval, "IDPortenClient should not exist")
}

func resourceExists(key client.ObjectKey, instance runtime.Object) func() bool {
	return func() bool {
		err := cli.Get(context.Background(), key, instance)
		return !errors.IsNotFound(err)
	}
}

func resourceDoesNotExist(key client.ObjectKey, instance runtime.Object) func() bool {
	return func() bool {
		err := cli.Get(context.Background(), key, instance)
		return errors.IsNotFound(err)
	}
}

func assertSecretExists(t *testing.T, name string, namespace string, instance *v1.IDPortenClient) {
	key := client.ObjectKey{
		Namespace: namespace,
		Name:      name,
	}
	a := &corev1.Secret{}
	err := cli.Get(context.Background(), key, a)
	assert.NoError(t, err)

	assert.True(t, containsOwnerRef(a.GetOwnerReferences(), *instance), "Secret should contain ownerReference")

	actualLabels := a.GetLabels()
	expectedLabels := map[string]string{
		labels.AppLabelKey:  instance.GetName(),
		labels.TypeLabelKey: labels.IDPortenTypeLabelValue,
	}
	assert.NotEmpty(t, actualLabels, "Labels should not be empty")
	assert.Equal(t, expectedLabels, actualLabels, "Labels should be set")

	assert.Equal(t, corev1.SecretTypeOpaque, a.Type, "Secret type should be Opaque")
	assert.NotEmpty(t, a.Data[secrets.IDPortenClientID])
	assert.NotEmpty(t, a.Data[secrets.IDPortenJwkKey])
	assert.NotEmpty(t, a.Data[secrets.IDPortenRedirectURI])
	assert.NotEmpty(t, a.Data[secrets.IDPortenWellKnownURL])

	assert.Empty(t, a.Data[secrets.MaskinportenJwkKey])
	assert.Empty(t, a.Data[secrets.MaskinportenClientID])
	assert.Empty(t, a.Data[secrets.MaskinportenScopes])
	assert.Empty(t, a.Data[secrets.MaskinportenWellKnownURL])
}

func containsOwnerRef(refs []metav1.OwnerReference, owner v1.IDPortenClient) bool {
	expected := metav1.OwnerReference{
		APIVersion: owner.APIVersion,
		Kind:       owner.Kind,
		Name:       owner.Name,
		UID:        owner.UID,
	}
	for _, ref := range refs {
		sameApiVersion := ref.APIVersion == expected.APIVersion
		sameKind := ref.Kind == expected.Kind
		sameName := ref.Name == expected.Name
		sameUID := ref.UID == expected.UID
		if sameApiVersion && sameKind && sameName && sameUID {
			return true
		}
	}
	return false
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
