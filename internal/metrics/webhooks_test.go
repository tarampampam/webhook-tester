package metrics_test

import (
	"testing"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/testutil"
	"github.com/stretchr/testify/assert"

	"gh.tarampamp.am/webhook-tester/internal/metrics"
)

func TestWebHooks_Register(t *testing.T) {
	var (
		registry = prometheus.NewRegistry()
		wh       = metrics.NewWebhooks()
	)

	assert.NoError(t, wh.Register(registry))

	count, err := testutil.GatherAndCount(registry, "webhooks_processed_count")
	assert.NoError(t, err)

	assert.Equal(t, 1, count)
}

func TestWebHooks_IncrementProcessedWebHooks(t *testing.T) {
	wh := metrics.NewWebhooks()

	wh.IncrementProcessedWebHooks()

	metric := getMetric(&wh, "webhooks_processed_count")
	assert.Equal(t, float64(1), metric.Counter.GetValue())
}
