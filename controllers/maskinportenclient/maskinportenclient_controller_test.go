package maskinportenclient_test

import (
	"context"
	"fmt"
	"os"
	"testing"

	nais_io_v1 "github.com/nais/liberator/pkg/apis/nais.io/v1"
	"github.com/stretchr/testify/assert"
	corev1 "k8s.io/api/core/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"

	"github.com/nais/digdirator/controllers/common"
	"github.com/nais/digdirator/controllers/common/test"
	"github.com/nais/digdirator/pkg/clients"
	"github.com/nais/digdirator/pkg/fixtures"
	"github.com/nais/digdirator/pkg/secrets"
	// +kubebuilder:scaffold:imports
)

var cli client.Client

func TestMain(m *testing.M) {
	testEnv, testEnvClient, err := test.SetupTestEnv(test.ClientID, test.MaskinportenHandlerType)
	if err != nil {
		os.Exit(1)
	}
	cli = *testEnvClient
	code := m.Run()
	_ = testEnv.Stop()
	os.Exit(code)
}

func TestMaskinportenController(t *testing.T) {
	cfg := fixtures.Config{
		DigdirClientName: "test-client",
		NamespaceName:    "test-namespace",
		SecretName:       "test-secret",
		UnusedSecretName: "test-unused-secret",
	}

	// set up preconditions for cluster
	clusterFixtures := fixtures.New(cli, cfg).MinimalConfig(clients.MaskinportenTypeLabelValue).WithNamespace()
	// create MaskinportenClient
	if err := clusterFixtures.Setup(); err != nil {
		t.Fatalf("failed to set up cluster fixtures: %v", err)
	}

	instance := &nais_io_v1.MaskinportenClient{}
	key := client.ObjectKey{
		Name:      "test-client",
		Namespace: "test-namespace",
	}
	assert.Eventually(t, test.ResourceExists(cli, key, instance), test.Timeout, test.Interval, "MaskinportenClient should exist")
	assert.Eventually(t, func() bool {
		err := cli.Get(context.Background(), key, instance)
		assert.NoError(t, err)
		b, err := clients.IsUpToDate(instance)
		assert.NoError(t, err)
		return b
	}, test.Timeout, test.Interval, "MaskinportenClient should be synchronized")
	assert.NotEmpty(t, instance.Status.ClientID)
	assert.NotEmpty(t, instance.Status.KeyIDs)
	assert.NotEmpty(t, instance.Status.CorrelationID)
	assert.NotEmpty(t, instance.Status.SynchronizationHash)
	assert.NotEmpty(t, instance.Status.SynchronizationTime)
	assert.Equal(t, common.EventSynchronized, instance.Status.SynchronizationState)

	assert.Equal(t, test.ClientID, instance.Status.ClientID)
	assert.Contains(t, instance.Status.KeyIDs, "some-keyid")
	assert.Len(t, instance.Status.KeyIDs, 1)

	secretAssertions := secretAssertions(t)
	test.AssertSecretExists(t, cli, cfg.SecretName, cfg.NamespaceName, instance, secretAssertions)

	assert.Eventually(t, test.ResourceDoesNotExist(cli, client.ObjectKey{
		Namespace: cfg.NamespaceName,
		Name:      cfg.UnusedSecretName,
	}, &corev1.Secret{}), test.Timeout, test.Interval, "unused Secret should not exist")

	// update MaskinportenClient
	previousSecretName := cfg.SecretName
	previousHash := instance.Status.SynchronizationHash
	previousCorrelationID := instance.Status.CorrelationID

	// set new secretname in spec -> trigger update
	instance.Spec.SecretName = "new-secret-name"
	err := cli.Update(context.Background(), instance)
	assert.NoError(t, err)

	// new hash should be set in status
	assert.Eventually(t, func() bool {
		err := cli.Get(context.Background(), key, instance)
		assert.NoError(t, err)
		return previousHash != instance.Status.SynchronizationHash
	}, test.Timeout, test.Interval, "new hash should be set")

	assert.Equal(t, test.ClientID, instance.Status.ClientID, "client ID should still match")
	assert.Len(t, instance.Status.KeyIDs, 2, "should contain two key IDs")
	assert.Contains(t, instance.Status.KeyIDs, "some-keyid", "previous key should still be valid")
	assert.Contains(t, instance.Status.KeyIDs, "some-new-keyid", "new key should be valid")
	assert.NotEqual(t, previousCorrelationID, instance.Status.CorrelationID, "should generate new correlation ID")
	assert.NotEmpty(t, instance.Status.SynchronizationHash)
	assert.NotEmpty(t, instance.Status.SynchronizationTime)
	assert.Equal(t, common.EventSynchronized, instance.Status.SynchronizationState)

	// new secret should exist
	test.AssertSecretExists(t, cli, instance.Spec.SecretName, cfg.NamespaceName, instance, secretAssertions)

	// old secret should still exist
	test.AssertSecretExists(t, cli, previousSecretName, cfg.NamespaceName, instance, secretAssertions)

	// delete MaskinportenClient
	err = cli.Delete(context.Background(), instance)

	assert.NoError(t, err, "deleting MaskinportenClient")

	assert.Eventually(t, test.ResourceDoesNotExist(cli, key, instance), test.Timeout, test.Interval, "MaskinportenClient should not exist")
}

func secretAssertions(t *testing.T) func(*corev1.Secret, clients.Instance) {
	return func(actual *corev1.Secret, instance clients.Instance) {
		actualLabels := actual.GetLabels()
		expectedLabels := map[string]string{
			clients.AppLabelKey:  instance.GetName(),
			clients.TypeLabelKey: clients.MaskinportenTypeLabelValue,
		}
		assert.NotEmpty(t, actualLabels, "Labels should not be empty")
		assert.Equal(t, expectedLabels, actualLabels, "Labels should be set")

		assert.Equal(t, corev1.SecretTypeOpaque, actual.Type, "Secret type should be Opaque")
		assert.NotEmpty(t, actual.Data[secrets.MaskinportenJwkKey])
		assert.NotEmpty(t, actual.Data[secrets.MaskinportenClientIDKey])
		assert.NotEmpty(t, actual.Data[secrets.MaskinportenScopesKey])
		assert.NotEmpty(t, actual.Data[secrets.MaskinportenWellKnownURLKey])

		assert.Empty(t, actual.Data[secrets.IDPortenClientIDKey])
		assert.Empty(t, actual.Data[secrets.IDPortenJwkKey])
		assert.Empty(t, actual.Data[secrets.IDPortenRedirectURIKey])
		assert.Empty(t, actual.Data[secrets.IDPortenWellKnownURLKey])
	}
}

func TestReconciler_CreateMaskinportenClient_ShouldNotProcessInSharedNamespace(t *testing.T) {
	appName := "shared-namespace-maskinporten"
	sharedNamespace := "shared"
	secretName := fmt.Sprintf("%s-%s", appName, test.AlreadyInUseSecret)
	cfg := fixtures.Config{
		DigdirClientName: appName,
		SecretName:       secretName,
		UnusedSecretName: test.UnusedSecret,
		NamespaceName:    sharedNamespace,
	}

	clusterFixtures := fixtures.New(cli, cfg).MinimalConfig(clients.MaskinportenTypeLabelValue).WithSharedNamespace()

	if err := clusterFixtures.Setup(); err != nil {
		t.Fatalf("failed to set up cluster fixtures: %v", err)
	}
	key := client.ObjectKey{
		Name:      appName,
		Namespace: sharedNamespace,
	}
	test.AssertApplicationShouldNotProcess(t, cli, key, &nais_io_v1.MaskinportenClient{})
}
