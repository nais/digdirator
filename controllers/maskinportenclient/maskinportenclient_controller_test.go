package maskinportenclient_test

import (
	"context"
	v1 "github.com/nais/digdirator/api/v1"
	"github.com/nais/digdirator/controllers/common/test"
	"github.com/nais/digdirator/pkg/fixtures"
	"github.com/nais/digdirator/pkg/labels"
	"github.com/stretchr/testify/assert"
	corev1 "k8s.io/api/core/v1"
	"os"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"testing"
	// +kubebuilder:scaffold:imports
)

const (
	clientID = "some-random-id"
)

var cli client.Client

func TestMain(m *testing.M) {
	testEnv, testEnvClient, err := test.SetupTestEnv(clientID, test.MaskinportenHandlerType)
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
	clusterFixtures := fixtures.New(cli, cfg).MinimalConfig().WithMaskinportenClient().WithPods().WithUnusedSecret(labels.MaskinportenTypeLabelValue)

	// create MaskinportenClient
	if err := clusterFixtures.Setup(); err != nil {
		t.Fatalf("failed to set up cluster fixtures: %v", err)
	}

	instance := &v1.MaskinportenClient{}
	key := client.ObjectKey{
		Name:      "test-client",
		Namespace: "test-namespace",
	}
	assert.Eventually(t, test.ResourceExists(cli, key, instance), test.Timeout, test.Interval, "MaskinportenClient should exist")
	assert.Eventually(t, func() bool {
		err := cli.Get(context.Background(), key, instance)
		assert.NoError(t, err)
		b, err := instance.IsUpToDate()
		assert.NoError(t, err)
		return b
	}, test.Timeout, test.Interval, "MaskinportenClient should be synchronized")
	assert.NotEmpty(t, instance.Status.ClientID)
	assert.NotEmpty(t, instance.Status.KeyIDs)
	assert.NotEmpty(t, instance.Status.CorrelationID)
	assert.NotEmpty(t, instance.Status.SynchronizationHash)
	assert.NotEmpty(t, instance.Status.SynchronizationTime)
	assert.Equal(t, v1.EventSynchronized, instance.Status.SynchronizationState)

	assert.Equal(t, clientID, instance.Status.ClientID)
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

	assert.Equal(t, clientID, instance.Status.ClientID, "client ID should still match")
	assert.Len(t, instance.Status.KeyIDs, 2, "should contain two key IDs")
	assert.Contains(t, instance.Status.KeyIDs, "some-keyid", "previous key should still be valid")
	assert.Contains(t, instance.Status.KeyIDs, "some-new-keyid", "new key should be valid")
	assert.NotEqual(t, previousCorrelationID, instance.Status.CorrelationID, "should generate new correlation ID")
	assert.NotEmpty(t, instance.Status.SynchronizationHash)
	assert.NotEmpty(t, instance.Status.SynchronizationTime)
	assert.Equal(t, v1.EventSynchronized, instance.Status.SynchronizationState)

	// new secret should exist
	test.AssertSecretExists(t, cli, instance.Spec.SecretName, cfg.NamespaceName, instance, secretAssertions)

	// old secret should still exist
	test.AssertSecretExists(t, cli, previousSecretName, cfg.NamespaceName, instance, secretAssertions)

	// delete MaskinportenClient
	err = cli.Delete(context.Background(), instance)

	assert.NoError(t, err, "deleting MaskinportenClient")

	assert.Eventually(t, test.ResourceDoesNotExist(cli, key, instance), test.Timeout, test.Interval, "MaskinportenClient should not exist")
}

func secretAssertions(t *testing.T) func(*corev1.Secret, v1.Instance) {
	return func(actual *corev1.Secret, instance v1.Instance) {
		actualLabels := actual.GetLabels()
		expectedLabels := map[string]string{
			labels.AppLabelKey:  instance.GetName(),
			labels.TypeLabelKey: labels.MaskinportenTypeLabelValue,
		}
		assert.NotEmpty(t, actualLabels, "Labels should not be empty")
		assert.Equal(t, expectedLabels, actualLabels, "Labels should be set")

		assert.Equal(t, corev1.SecretTypeOpaque, actual.Type, "Secret type should be Opaque")
		assert.NotEmpty(t, actual.Data[v1.MaskinportenJwkKey])
		assert.NotEmpty(t, actual.Data[v1.MaskinportenClientID])
		assert.NotEmpty(t, actual.Data[v1.MaskinportenScopes])
		assert.NotEmpty(t, actual.Data[v1.MaskinportenWellKnownURL])

		assert.Empty(t, actual.Data[v1.IDPortenClientID])
		assert.Empty(t, actual.Data[v1.IDPortenJwkKey])
		assert.Empty(t, actual.Data[v1.IDPortenRedirectURI])
		assert.Empty(t, actual.Data[v1.IDPortenWellKnownURL])
	}
}
