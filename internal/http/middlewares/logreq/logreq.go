// Package logreq contains middleware for HTTP requests logging using "zap" package.
package logreq

import (
	"strings"

	"github.com/labstack/echo/v4"
	"github.com/labstack/echo/v4/middleware"
	"go.uber.org/zap"
)

// New creates echo.MiddlewareFunc for HTTP requests logging using "zap" package.
func New(log *zap.Logger, dbgRoutesPrefixes []string) echo.MiddlewareFunc {
	return middleware.RequestLoggerWithConfig(middleware.RequestLoggerConfig{
		LogUserAgent: true,
		LogMethod:    true,
		LogStatus:    true,
		LogLatency:   true,

		LogValuesFunc: func(c echo.Context, v middleware.RequestLoggerValues) error {
			var logLvl = zap.InfoLevel // default logging level

			// log the following routes only for debug level
			for _, s := range dbgRoutesPrefixes {
				if strings.HasPrefix(c.Request().URL.Path, s) {
					logLvl = zap.DebugLevel

					break
				}
			}

			if ce := log.Check(logLvl, "HTTP request processed"); ce != nil {
				ce.Write(
					zap.String("remote addr", c.RealIP()),
					zap.String("useragent", v.UserAgent),
					zap.String("method", v.Method),
					zap.String("uri", c.Request().URL.String()),
					zap.Int("status code", v.Status),
					zap.Duration("duration", v.Latency),
				)
			}

			return nil
		},
	})
}
