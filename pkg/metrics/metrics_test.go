package metrics

import (
	"testing"

	"github.com/nais/digdirator/pkg/config"
	"github.com/stretchr/testify/assert"
)

func TestMetricsForFeatures(t *testing.T) {
	assert.Empty(t, AllMetricsForFeatures(config.Features{}))
	assert.ElementsMatch(t, idPortenMetrics, AllMetricsForFeatures(config.Features{IDPorten: true}))
	assert.ElementsMatch(t, ansattportenMetrics, AllMetricsForFeatures(config.Features{Ansattporten: true}))
	assert.ElementsMatch(t, maskinportenMetrics, AllMetricsForFeatures(config.Features{Maskinporten: true}))
}

func TestCountersForFeatures(t *testing.T) {
	assert.Empty(t, AllCountersForFeatures(config.Features{}))
	assert.Len(t, AllCountersForFeatures(config.Features{IDPorten: true}), 7)
	assert.Len(t, AllCountersForFeatures(config.Features{Ansattporten: true}), 7)
	assert.Len(t, AllCountersForFeatures(config.Features{Maskinporten: true}), 13)
}
