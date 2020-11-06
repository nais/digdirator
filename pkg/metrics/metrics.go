package metrics

import (
	"context"
	v1 "github.com/nais/digdirator/api/v1"
	"github.com/nais/digdirator/pkg/labels"
	"github.com/nais/digdirator/pkg/namespaces"
	"github.com/prometheus/client_golang/prometheus"
	log "github.com/sirupsen/logrus"
	corev1 "k8s.io/api/core/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"time"
)

const (
	labelNamespace = "namespace"
)

var (
	IDPortenClientsTotal = prometheus.NewGauge(
		prometheus.GaugeOpts{
			Name: "idporten_client_total",
		})
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
	IDPortenClientsDeletedCount = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Name: "idporten_client_deleted_count",
			Help: "Number of idporten clients successfully deleted",
		},
		[]string{labelNamespace},
	)
)

var AllMetrics = []prometheus.Collector{
	IDPortenClientsTotal,
	IDPortenSecretsTotal,
	IDPortenClientsProcessedCount,
	IDPortenClientsFailedProcessingCount,
	IDPortenClientsCreatedCount,
	IDPortenClientsUpdatedCount,
	IDPortenClientsRotatedCount,
	IDPortenClientsDeletedCount,
}

var AllCounters = []*prometheus.CounterVec{
	IDPortenClientsProcessedCount,
	IDPortenClientsFailedProcessingCount,
	IDPortenClientsCreatedCount,
	IDPortenClientsUpdatedCount,
	IDPortenClientsRotatedCount,
	IDPortenClientsDeletedCount,
}

func IncWithNamespaceLabel(metric *prometheus.CounterVec, namespace string) {
	metric.WithLabelValues(namespace).Inc()
}

type Metrics interface {
	Refresh(ctx context.Context)
}

type metrics struct {
	reader client.Reader
}

func New(reader client.Reader) Metrics {
	return metrics{
		reader: reader,
	}
}

func (m metrics) InitWithNamespaceLabels() {
	ns, err := namespaces.GetAll(context.Background(), m.reader)
	if err != nil {
		log.Errorf("failed to list namespaces: %v", err)
	}
	for _, n := range ns.Items {
		for _, c := range AllCounters {
			c.WithLabelValues(n.Name).Add(0)
		}
	}
}

func (m metrics) Refresh(ctx context.Context) {
	var err error
	exp := 10 * time.Second

	mLabels := client.MatchingLabels{
		labels.TypeLabelKey: labels.IDPortenTypeLabelValue,
	}

	var secretList corev1.SecretList
	var idportenClientsList v1.IDPortenClientList

	m.InitWithNamespaceLabels()

	t := time.NewTicker(exp)
	for range t.C {
		log.Debug("Refreshing metrics from cluster")
		if err = m.reader.List(ctx, &secretList, mLabels); err != nil {
			log.Errorf("failed to list secrets: %v", err)
		}
		IDPortenSecretsTotal.Set(float64(len(secretList.Items)))

		if err = m.reader.List(ctx, &idportenClientsList); err != nil {
			log.Errorf("failed to list idporten clients: %v", err)
		}
		IDPortenClientsTotal.Set(float64(len(idportenClientsList.Items)))
	}
}
