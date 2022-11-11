// Package logreq contains middleware for HTTP requests logging using "zap" package.
package logreq

import (
	"net/http"
	"strings"

	"github.com/felixge/httpsnoop"
	"go.uber.org/zap"

	"github.com/tarampampam/webhook-tester/internal/api"
	"github.com/tarampampam/webhook-tester/internal/pkg/realip"
)

// New creates mux.MiddlewareFunc for HTTP requests logging using "zap" package.
func New(log *zap.Logger) api.MiddlewareFunc {
	return func(next http.HandlerFunc) http.HandlerFunc {
		return func(w http.ResponseWriter, r *http.Request) {
			metrics := httpsnoop.CaptureMetrics(next, w, r)

			if !strings.Contains(strings.ToLower(r.UserAgent()), "healthcheck") {
				log.Info("HTTP request processed",
					zap.String("remote addr", realip.FromHTTPRequest(r)),
					zap.String("useragent", r.UserAgent()),
					zap.String("method", r.Method),
					zap.String("url", r.URL.String()),
					zap.Int("status code", metrics.Code),
					zap.Duration("duration", metrics.Duration),
				)
			}
		}
	}
}
