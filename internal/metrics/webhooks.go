package metrics

import "github.com/prometheus/client_golang/prometheus"

type WebHooks struct {
	processedCounter prometheus.Counter
}

// NewWebhooks creates new WebHooks metrics collector.
func NewWebhooks() WebHooks {
	return WebHooks{
		processedCounter: prometheus.NewCounter(prometheus.CounterOpts{ //nolint:promlinter
			Namespace: "webhooks",
			Subsystem: "processed",
			Name:      "count",
			Help:      "The count of processed webhooks.",
		}),
	}
}

// IncrementProcessedWebHooks increments processed webhooks counter.
func (w *WebHooks) IncrementProcessedWebHooks() { w.processedCounter.Inc() }

// Register metrics with registerer.
func (w *WebHooks) Register(reg prometheus.Registerer) error {
	if e := reg.Register(w.processedCounter); e != nil {
		return e
	}

	return nil
}
