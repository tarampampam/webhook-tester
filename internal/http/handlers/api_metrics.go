package handlers

import (
	"github.com/labstack/echo/v4"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

type apiMetrics struct {
	registry prometheus.Gatherer
}

func (s *apiMetrics) AppMetrics(c echo.Context) error {
	return echo.WrapHandler(
		promhttp.HandlerFor(
			s.registry,
			promhttp.HandlerOpts{ErrorHandling: promhttp.ContinueOnError},
		),
	)(c)
}
