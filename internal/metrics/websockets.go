package metrics

import "github.com/prometheus/client_golang/prometheus"

type WebSockets struct {
	clientsCounter prometheus.Gauge
}

// NewWebsockets creates new WebSockets metrics collector.
func NewWebsockets() WebSockets {
	return WebSockets{
		clientsCounter: prometheus.NewGauge(prometheus.GaugeOpts{ //nolint:promlinter
			Namespace: "websockets",
			Subsystem: "active_clients",
			Name:      "count",
			Help:      "The count of active websocket clients.",
		}),
	}
}

// IncrementActiveClients increments active websocket clients count.
func (w *WebSockets) IncrementActiveClients() { w.clientsCounter.Inc() }

// DecrementActiveClients decrements active websocket clients count.
func (w *WebSockets) DecrementActiveClients() { w.clientsCounter.Dec() }

// Register metrics with registerer.
func (w *WebSockets) Register(reg prometheus.Registerer) error {
	if e := reg.Register(w.clientsCounter); e != nil {
		return e
	}

	return nil
}
