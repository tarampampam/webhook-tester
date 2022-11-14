package handlers_test

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/labstack/echo/v4"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/stretchr/testify/assert"

	"github.com/tarampampam/webhook-tester/internal/config"
	"github.com/tarampampam/webhook-tester/internal/http/handlers"
)

func TestApiMetrics_AppMetrics(t *testing.T) {
	var (
		registry   = prometheus.NewRegistry()
		testMetric = prometheus.NewGaugeVec(
			prometheus.GaugeOpts{
				Namespace: "foo",
				Subsystem: "bar",
				Name:      "test",
				Help:      "Test metric.",
			},
			[]string{"foo"},
		)
	)

	registry.MustRegister(testMetric)
	testMetric.WithLabelValues("bar").Set(1)

	var (
		api = handlers.NewAPI(
			context.Background(), config.Config{}, nil, nil, nil, nil, registry, "", nil,
		)

		req, _ = http.NewRequest(http.MethodGet, "http://test/", http.NoBody)
		rr     = httptest.NewRecorder()
	)

	assert.NoError(t, api.AppMetrics(echo.New().NewContext(req, rr)))

	assert.Equal(t, rr.Code, http.StatusOK)
	assert.Equal(t, 200, rr.Code)
	assert.Equal(t, `# HELP foo_bar_test Test metric.
# TYPE foo_bar_test gauge
foo_bar_test{foo="bar"} 1
`, rr.Body.String())
	assert.Regexp(t, "^text/plain.*$", rr.Header().Get("Content-Type"))
}
