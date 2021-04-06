package metrics_test

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/stretchr/testify/assert"
	"github.com/tarampampam/webhook-tester/internal/pkg/http/handlers/metrics"
)

func TestNewHandlerError(t *testing.T) {
	var (
		req, _     = http.NewRequest(http.MethodGet, "http://testing?foo=bar", http.NoBody)
		rr         = httptest.NewRecorder()
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

	metrics.NewHandler(registry)(rr, req)

	assert.Equal(t, rr.Code, http.StatusOK)
	assert.Equal(t, 200, rr.Code)
	assert.Equal(t, `# HELP foo_bar_test Test metric.
# TYPE foo_bar_test gauge
foo_bar_test{foo="bar"} 1
`, rr.Body.String())
	assert.Regexp(t, "^text/plain.*$", rr.Header().Get("Content-Type"))
}
