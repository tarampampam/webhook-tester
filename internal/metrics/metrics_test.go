package metrics_test

import (
	"testing"

	"github.com/prometheus/client_golang/prometheus/testutil"
	"github.com/stretchr/testify/assert"

	"gh.tarampamp.am/webhook-tester/internal/metrics"
)

func TestNewRegistry(t *testing.T) {
	registry := metrics.NewRegistry()

	count, err := testutil.GatherAndCount(registry)

	assert.NoError(t, err)
	assert.True(t, count >= 30, "not enough common metrics")
}
