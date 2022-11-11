// Package logreq contains middleware for HTTP requests logging using "zap" package.
package logreq

import (
	"github.com/labstack/echo/v4"
	"github.com/labstack/echo/v4/middleware"
	"go.uber.org/zap"

	"github.com/tarampampam/webhook-tester/internal/pkg/realip"
)

// New creates echo.MiddlewareFunc for HTTP requests logging using "zap" package.
func New(log *zap.Logger) echo.MiddlewareFunc {
	return middleware.RequestLoggerWithConfig(middleware.RequestLoggerConfig{
		LogUserAgent: true,
		LogMethod:    true,
		LogURI:       true,
		LogStatus:    true,
		LogLatency:   true,

		LogValuesFunc: func(c echo.Context, v middleware.RequestLoggerValues) error {
			log.Info("HTTP request processed",
				zap.String("remote addr", realip.FromHTTPRequest(c.Request())),
				zap.String("useragent", v.UserAgent),
				zap.String("method", v.Method),
				zap.String("uri", v.URI),
				zap.Int("status code", v.Status),
				zap.Duration("duration", v.Latency),
			)

			return nil
		},
	})
}
