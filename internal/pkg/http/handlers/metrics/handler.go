// Package metrics contains HTTP handler for application metrics (prometheus format) generation.
package metrics

import (
	"net/http"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

// NewHandler creates metrics handler.
func NewHandler(registry prometheus.Gatherer) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		promhttp.HandlerFor(registry, promhttp.HandlerOpts{ErrorHandling: promhttp.ContinueOnError}).ServeHTTP(w, r)
	}
}
