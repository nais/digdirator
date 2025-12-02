package metrics

import (
	"context"
	"fmt"
	"log/slog"
	"time"

	naisiov1 "github.com/nais/liberator/pkg/apis/nais.io/v1"
	"github.com/nais/liberator/pkg/kubernetes"
	"github.com/prometheus/client_golang/prometheus"
	corev1 "k8s.io/api/core/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"

	"github.com/nais/digdirator/pkg/clients"
	"github.com/nais/digdirator/pkg/digdir/types"
	"github.com/nais/digdirator/pkg/retry"
)

const (
	labelNamespace = "namespace"
)

var log *slog.Logger

var (
	IDPortenClientsTotal = prometheus.NewGauge(
		prometheus.GaugeOpts{
			Name: "idporten_client_total",
		},
	)
	IDPortenSecretsTotal = prometheus.NewGauge(
		prometheus.GaugeOpts{
			Name: "idporten_client_secrets_total",
			Help: "Total number of idporten client secrets",
		},
	)
	IDPortenClientsCreatedCount = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Name: "idporten_client_created_count",
			Help: "Number of idporten clients created successfully",
		},
		[]string{labelNamespace},
	)
	IDPortenClientsUpdatedCount = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Name: "idporten_client_updated_count",
			Help: "Number of idporten clients updated successfully",
		},
		[]string{labelNamespace},
	)
	IDPortenClientsRotatedCount = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Name: "idporten_client_rotated_count",
			Help: "Number of idporten clients successfully rotated credentials",
		},
		[]string{labelNamespace},
	)
	IDPortenClientsProcessedCount = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Name: "idporten_client_processed_count",
			Help: "Number of idporten clients processed successfully",
		},
		[]string{labelNamespace},
	)
	IDPortenClientsFailedProcessingCount = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Name: "idporten_client_failed_processing_count",
			Help: "Number of idporten clients that failed processing",
		},
		[]string{labelNamespace},
	)
	IDPortenClientsFailedInvalidConfigCount = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Name: "idporten_client_failed_invalid_config_count",
			Help: "Number of idporten clients that failed processing due to invalid configuration",
		},
		[]string{labelNamespace},
	)
	IDPortenClientsDeletedCount = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Name: "idporten_client_deleted_count",
			Help: "Number of idporten clients successfully deleted",
		},
		[]string{labelNamespace},
	)
	MaskinportenClientsTotal = prometheus.NewGauge(
		prometheus.GaugeOpts{
			Name: "maskinporten_client_total",
		},
	)
	MaskinportenSecretsTotal = prometheus.NewGauge(
		prometheus.GaugeOpts{
			Name: "maskinporten_client_secrets_total",
			Help: "Total number of maskinporten client secrets",
		},
	)
	MaskinportenClientsCreatedCount = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Name: "maskinporten_client_created_count",
			Help: "Number of maskinporten clients created successfully",
		},
		[]string{labelNamespace},
	)
	MaskinportenClientsUpdatedCount = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Name: "maskinporten_client_updated_count",
			Help: "Number of maskinporten clients updated successfully",
		},
		[]string{labelNamespace},
	)
	MaskinportenClientsRotatedCount = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Name: "maskinporten_client_rotated_count",
			Help: "Number of maskinporten clients successfully rotated credentials",
		},
		[]string{labelNamespace},
	)
	MaskinportenClientsProcessedCount = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Name: "maskinporten_client_processed_count",
			Help: "Number of maskinporten clients processed successfully",
		},
		[]string{labelNamespace},
	)
	MaskinportenClientsFailedProcessingCount = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Name: "maskinporten_client_failed_processing_count",
			Help: "Number of maskinporten clients that failed processing",
		},
		[]string{labelNamespace},
	)
	MaskinportenClientsFailedInvalidConfigCount = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Name: "maskinporten_client_failed_invalid_config_count",
			Help: "Number of maskinporten clients that failed processing due to invalid configuration",
		},
		[]string{labelNamespace},
	)
	MaskinportenClientsDeletedCount = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Name: "maskinporten_client_deleted_count",
			Help: "Number of maskinporten clients successfully deleted",
		},
		[]string{labelNamespace},
	)
	MaskinportenExposedScopesTotal = prometheus.NewGauge(
		prometheus.GaugeOpts{
			Name: "maskinporten_exposed_scope_total",
		},
	)
	MaskinportenExternalScopesConsumedTotal = prometheus.NewGauge(
		prometheus.GaugeOpts{
			Name: "maskinporten_external_scopes_consumed_total",
		},
	)
	MaskinportenScopeConsumersTotal = prometheus.NewGauge(
		prometheus.GaugeOpts{
			Name: "maskinporten_scope_consumers_total",
		},
	)
	MaskinportenScopesCreatedCount = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Name: "maskinporten_scope_created_count",
			Help: "Number of maskinporten scopes successfully created",
		},
		[]string{labelNamespace},
	)
	MaskinportenScopesUpdatedCount = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Name: "maskinporten_scope_updated_count",
			Help: "Number of maskinporten scopes successfully updated",
		},
		[]string{labelNamespace},
	)
	MaskinportenScopesDeletedCount = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Name: "maskinporten_scope_deleted_count",
			Help: "Number of maskinporten scopes successfully deleted",
		},
		[]string{labelNamespace},
	)
	MaskinportenScopesReactivatedCount = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Name: "maskinporten_scope_reactivated_count",
			Help: "Number of maskinporten scopes successfully reactivated",
		},
		[]string{labelNamespace},
	)
	MaskinportenScopesConsumersCreatedCount = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Name: "maskinporten_scope_consumer_created_count",
			Help: "Number of maskinporten scope consumers successfully created",
		},
		[]string{labelNamespace},
	)
	MaskinportenScopesConsumersUpdatedCount = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Name: "maskinporten_scope_consumer_updated_count",
			Help: "Number of maskinporten scope consumers successfully updated",
		},
		[]string{labelNamespace},
	)
	MaskinportenScopesConsumersDeletedCount = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Name: "maskinporten_scope_consumer_deleted_count",
			Help: "Number of maskinporten scope consumers successfully deleted",
		},
		[]string{labelNamespace},
	)
)

var AllMetrics = []prometheus.Collector{
	IDPortenClientsTotal,
	IDPortenSecretsTotal,
	IDPortenClientsProcessedCount,
	IDPortenClientsFailedProcessingCount,
	IDPortenClientsFailedInvalidConfigCount,
	IDPortenClientsCreatedCount,
	IDPortenClientsUpdatedCount,
	IDPortenClientsRotatedCount,
	IDPortenClientsDeletedCount,
	MaskinportenClientsTotal,
	MaskinportenSecretsTotal,
	MaskinportenClientsProcessedCount,
	MaskinportenClientsFailedProcessingCount,
	MaskinportenClientsCreatedCount,
	MaskinportenClientsUpdatedCount,
	MaskinportenClientsRotatedCount,
	MaskinportenClientsDeletedCount,
	MaskinportenExposedScopesTotal,
	MaskinportenExternalScopesConsumedTotal,
	MaskinportenScopeConsumersTotal,
	MaskinportenScopesCreatedCount,
	MaskinportenScopesUpdatedCount,
	MaskinportenScopesDeletedCount,
	MaskinportenScopesReactivatedCount,
	MaskinportenScopesConsumersCreatedCount,
	MaskinportenScopesConsumersUpdatedCount,
	MaskinportenScopesConsumersDeletedCount,
}

var AllCounters = []*prometheus.CounterVec{
	IDPortenClientsProcessedCount,
	IDPortenClientsFailedProcessingCount,
	IDPortenClientsCreatedCount,
	IDPortenClientsUpdatedCount,
	IDPortenClientsRotatedCount,
	IDPortenClientsDeletedCount,
	MaskinportenClientsProcessedCount,
	MaskinportenClientsFailedProcessingCount,
	MaskinportenClientsFailedInvalidConfigCount,
	MaskinportenClientsCreatedCount,
	MaskinportenClientsUpdatedCount,
	MaskinportenClientsRotatedCount,
	MaskinportenClientsDeletedCount,
	MaskinportenScopesCreatedCount,
	MaskinportenScopesUpdatedCount,
	MaskinportenScopesDeletedCount,
	MaskinportenScopesReactivatedCount,
	MaskinportenScopesConsumersCreatedCount,
	MaskinportenScopesConsumersUpdatedCount,
	MaskinportenScopesConsumersDeletedCount,
}

func incWithNamespaceLabel(metric *prometheus.CounterVec, namespace string) {
	metric.WithLabelValues(namespace).Inc()
}

func IncClientsProcessed(instance clients.Instance) {
	switch instance.(type) {
	case *naisiov1.IDPortenClient:
		incWithNamespaceLabel(IDPortenClientsProcessedCount, instance.GetNamespace())
	case *naisiov1.MaskinportenClient:
		incWithNamespaceLabel(MaskinportenClientsProcessedCount, instance.GetNamespace())
	}
}

func IncClientsFailedProcessing(instance clients.Instance) {
	switch instance.(type) {
	case *naisiov1.IDPortenClient:
		incWithNamespaceLabel(IDPortenClientsFailedProcessingCount, instance.GetNamespace())
	case *naisiov1.MaskinportenClient:
		incWithNamespaceLabel(MaskinportenClientsFailedProcessingCount, instance.GetNamespace())
	}
}

func IncClientsFailedInvalidConfig(instance clients.Instance) {
	switch instance.(type) {
	case *naisiov1.IDPortenClient:
		incWithNamespaceLabel(IDPortenClientsFailedInvalidConfigCount, instance.GetNamespace())
	case *naisiov1.MaskinportenClient:
		incWithNamespaceLabel(MaskinportenClientsFailedInvalidConfigCount, instance.GetNamespace())
	}
}

func IncClientsCreated(instance clients.Instance) {
	switch instance.(type) {
	case *naisiov1.IDPortenClient:
		incWithNamespaceLabel(IDPortenClientsCreatedCount, instance.GetNamespace())
	case *naisiov1.MaskinportenClient:
		incWithNamespaceLabel(MaskinportenClientsCreatedCount, instance.GetNamespace())
	}
}

func IncClientsUpdated(instance clients.Instance) {
	switch instance.(type) {
	case *naisiov1.IDPortenClient:
		incWithNamespaceLabel(IDPortenClientsUpdatedCount, instance.GetNamespace())
	case *naisiov1.MaskinportenClient:
		incWithNamespaceLabel(MaskinportenClientsUpdatedCount, instance.GetNamespace())
	}
}

func IncClientsRotated(instance clients.Instance) {
	switch instance.(type) {
	case *naisiov1.IDPortenClient:
		incWithNamespaceLabel(IDPortenClientsRotatedCount, instance.GetNamespace())
	case *naisiov1.MaskinportenClient:
		incWithNamespaceLabel(MaskinportenClientsRotatedCount, instance.GetNamespace())
	}
}

func IncClientsDeleted(instance clients.Instance) {
	switch instance.(type) {
	case *naisiov1.IDPortenClient:
		incWithNamespaceLabel(IDPortenClientsDeletedCount, instance.GetNamespace())
	case *naisiov1.MaskinportenClient:
		incWithNamespaceLabel(MaskinportenClientsDeletedCount, instance.GetNamespace())
	}
}

func IncScopesCreated(instance clients.Instance) {
	switch instance.(type) {
	case *naisiov1.MaskinportenClient:
		incWithNamespaceLabel(MaskinportenScopesCreatedCount, instance.GetNamespace())
	}
}

func IncScopesUpdated(instance clients.Instance) {
	switch instance.(type) {
	case *naisiov1.MaskinportenClient:
		incWithNamespaceLabel(MaskinportenScopesUpdatedCount, instance.GetNamespace())
	}
}

func IncScopesDeleted(instance clients.Instance) {
	switch instance.(type) {
	case *naisiov1.MaskinportenClient:
		incWithNamespaceLabel(MaskinportenScopesDeletedCount, instance.GetNamespace())
	}
}

func IncScopesReactivated(instance clients.Instance) {
	switch instance.(type) {
	case *naisiov1.MaskinportenClient:
		incWithNamespaceLabel(MaskinportenScopesDeletedCount, instance.GetNamespace())
	}
}

func IncScopesConsumersCreatedOrUpdated(instance clients.Instance, state types.State) {
	switch instance.(type) {
	case *naisiov1.MaskinportenClient:
		if state == types.ScopeStateDenied {
			incWithNamespaceLabel(MaskinportenScopesConsumersUpdatedCount, instance.GetNamespace())
		} else {
			incWithNamespaceLabel(MaskinportenScopesConsumersCreatedCount, instance.GetNamespace())
		}
	}
}

func IncScopesConsumersDeleted(instance clients.Instance) {
	switch instance.(type) {
	case *naisiov1.MaskinportenClient:
		incWithNamespaceLabel(MaskinportenScopesConsumersDeletedCount, instance.GetNamespace())
	}
}

type Metrics interface {
	Refresh(ctx context.Context)
}

type metrics struct {
	reader client.Reader
}

func New(reader client.Reader) Metrics {
	log = slog.Default().With("subsystem", "metrics")
	return metrics{
		reader: reader,
	}
}

func (m metrics) InitWithNamespaceLabels() {
	var ns corev1.NamespaceList
	var err error

	retryable := func(ctx context.Context) error {
		ns, err = kubernetes.ListNamespaces(context.Background(), m.reader)
		if err != nil {
			return retry.RetryableError(fmt.Errorf("listing namespaces: %w", err))
		}
		return nil
	}

	err = retry.Fibonacci(1*time.Second).
		WithMaxDuration(1*time.Minute).
		Do(context.Background(), retryable)
	if err != nil {
		log.Error("listing namespaces", "error", err)
	}

	for _, n := range ns.Items {
		for _, c := range AllCounters {
			c.WithLabelValues(n.Name).Add(0)
		}
	}

	log.Info("metrics with namespace labels initialized")
}

func (m metrics) Refresh(ctx context.Context) {
	var err error
	exp := 10 * time.Second

	var idportenSecretList corev1.SecretList
	var idportenClientsList naisiov1.IDPortenClientList

	var maskinportenSecretList corev1.SecretList
	var maskinportenClientsList naisiov1.MaskinportenClientList

	m.InitWithNamespaceLabels()

	t := time.NewTicker(exp)
	for range t.C {
		if err = m.reader.List(ctx, &idportenSecretList, client.MatchingLabels{
			clients.TypeLabelKey: clients.IDPortenTypeLabelValue,
		}); err != nil {
			log.Error("failed to list idporten secrets", "error", err)
		}
		IDPortenSecretsTotal.Set(float64(len(idportenSecretList.Items)))

		if err = m.reader.List(ctx, &maskinportenSecretList, client.MatchingLabels{
			clients.TypeLabelKey: clients.MaskinportenTypeLabelValue,
		}); err != nil {
			log.Error("failed to list maskinporten secrets", "error", err)
		}
		MaskinportenSecretsTotal.Set(float64(len(maskinportenSecretList.Items)))

		if err = m.reader.List(ctx, &idportenClientsList); err != nil {
			log.Error("failed to list idporten clients", "error", err)
		}
		IDPortenClientsTotal.Set(float64(len(idportenClientsList.Items)))

		if err = m.reader.List(ctx, &maskinportenClientsList); err != nil {
			log.Error("failed to list maskinporten clients", "error", err)
		}
		MaskinportenClientsTotal.Set(float64(len(maskinportenClientsList.Items)))

		setTotalForMaskinportenScopes(maskinportenClientsList.Items)
	}
}

func setTotalForMaskinportenScopes(maskinportenClients []naisiov1.MaskinportenClient) {
	var exposedTotal int
	var consumedTotal int
	var consumersTotal int

	for _, c := range maskinportenClients {

		if c.Spec.Scopes.ConsumedScopes != nil {
			consumedTotal = consumedTotal + len(c.Spec.Scopes.ConsumedScopes)
		}

		if c.Spec.Scopes.ExposedScopes != nil {
			exposedTotal = exposedTotal + len(c.Spec.Scopes.ExposedScopes)
			setTotalConsumersOfScope(c, &consumersTotal)
		}
	}
	MaskinportenExposedScopesTotal.Set(float64(exposedTotal))
	MaskinportenExternalScopesConsumedTotal.Set(float64(consumedTotal))
	MaskinportenScopeConsumersTotal.Set(float64(consumersTotal))
}

func setTotalConsumersOfScope(client naisiov1.MaskinportenClient, consumersTotal *int) {
	for _, e := range client.Spec.Scopes.ExposedScopes {
		if e.Consumers != nil {
			*consumersTotal = *consumersTotal + len(e.Consumers)
		}
	}
}
