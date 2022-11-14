package metrics_test

import (
	"github.com/prometheus/client_golang/prometheus"
	dto "github.com/prometheus/client_model/go"
)

type registerer interface {
	Register(prometheus.Registerer) error
}

func getMetric(m registerer, name string) *dto.Metric {
	registry := prometheus.NewRegistry()
	_ = m.Register(registry)

	families, _ := registry.Gather()

	for _, family := range families {
		if family.GetName() == name {
			return family.Metric[0]
		}
	}

	return nil
}
