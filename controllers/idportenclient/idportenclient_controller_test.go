package idportenclient_test

import (
	"context"
	"log/slog"
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
	handler := test.IDPortenHandler(test.ClientID)
	testEnv, testEnvClient, err := test.SetupTestEnv(handler)
	if err != nil {
		slog.Error("setting up test environment", "error", err)
	}
	cli = *testEnvClient
	code := m.Run()
	_ = testEnv.Stop()
	os.Exit(code)
}

func TestIDPortenController(t *testing.T) {
	cfg := fixtures.Config{
		DigdirClientName: "test-client",
		NamespaceName:    "test-namespace",
		SecretName:       "test-secret",
		UnusedSecretName: "test-unused-secret",
	}

	// set up preconditions for cluster
	clusterFixtures := fixtures.New(cli, cfg).MinimalConfig(clients.IDPortenTypeLabelValue).WithNamespace()

	// create IDPortenClient
	if err := clusterFixtures.Setup(); err != nil {
		t.Fatalf("failed to set up cluster fixtures: %v", err)
	}

	instance := &nais_io_v1.IDPortenClient{}
	key := client.ObjectKey{
		Name:      "test-client",
		Namespace: "test-namespace",
	}
	assert.Eventually(t, test.ResourceExists(cli, key, instance), test.Timeout, test.Interval, "IDPortenClient should exist")
	assert.Eventually(t, func() bool {
		err := cli.Get(context.Background(), key, instance)
		assert.NoError(t, err)
		return clients.IsUpToDate(instance)
	}, test.Timeout, test.Interval, "IDPortenClient should be synchronized")
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

	// update IDPortenClient
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

	// delete IDPortenClient
	err = cli.Delete(context.Background(), instance)

	assert.NoError(t, err, "deleting IDPortenClient")

	assert.Eventually(t, test.ResourceDoesNotExist(cli, key, instance), test.Timeout, test.Interval, "IDPortenClient should not exist")
}

func secretAssertions(t *testing.T) func(*corev1.Secret, clients.Instance) {
	return func(actual *corev1.Secret, instance clients.Instance) {
		actualLabels := actual.GetLabels()
		expectedLabels := map[string]string{
			clients.AppLabelKey:  instance.GetName(),
			clients.TypeLabelKey: clients.IDPortenTypeLabelValue,
		}
		assert.NotEmpty(t, actualLabels, "Labels should not be empty")
		assert.Equal(t, expectedLabels, actualLabels, "Labels should be set")

		actualAnnotations := actual.GetAnnotations()
		expectedAnnotations := map[string]string{
			common.StakaterReloaderKeyAnnotation: "true",
		}
		assert.NotEmpty(t, actualAnnotations, "Annotations should not be empty")
		assert.Equal(t, expectedAnnotations, actualAnnotations, "Annotations should be set")

		assert.Equal(t, corev1.SecretTypeOpaque, actual.Type, "Secret type should be Opaque")
		assert.NotEmpty(t, actual.Data[secrets.IDPortenClientIDKey])
		assert.NotEmpty(t, actual.Data[secrets.IDPortenJwkKey])
		assert.NotEmpty(t, actual.Data[secrets.IDPortenRedirectURIKey])
		assert.NotEmpty(t, actual.Data[secrets.IDPortenWellKnownURLKey])
		assert.NotEmpty(t, actual.Data[secrets.IDPortenIssuerKey])
		assert.NotEmpty(t, actual.Data[secrets.IDPortenJwksUriKey])
		assert.NotEmpty(t, actual.Data[secrets.IDPortenTokenEndpointKey])

		assert.Empty(t, actual.Data[secrets.MaskinportenJwkKey])
		assert.Empty(t, actual.Data[secrets.MaskinportenClientIDKey])
		assert.Empty(t, actual.Data[secrets.MaskinportenScopesKey])
		assert.Empty(t, actual.Data[secrets.MaskinportenWellKnownURLKey])
		assert.Empty(t, actual.Data[secrets.MaskinportenIssuerKey])
		assert.Empty(t, actual.Data[secrets.MaskinportenJwksUriKey])
		assert.Empty(t, actual.Data[secrets.MaskinportenTokenEndpointKey])
	}
}
