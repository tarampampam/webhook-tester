package metrics_test

import (
	"testing"

	"github.com/prometheus/client_golang/prometheus/testutil"
	"github.com/stretchr/testify/assert"
	"github.com/tarampampam/webhook-tester/internal/pkg/metrics"
)

func TestNewRegistry(t *testing.T) {
	registry := metrics.NewRegistry()

	count, err := testutil.GatherAndCount(registry)

	assert.NoError(t, err)
	assert.True(t, count >= 35, "not enough common metrics")
}
