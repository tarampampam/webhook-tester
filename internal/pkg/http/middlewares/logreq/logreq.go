// Package logreq contains middleware for HTTP requests logging using "zap" package.
package logreq

import (
	"net/http"

	"github.com/felixge/httpsnoop"
	"github.com/gorilla/mux"
	"github.com/tarampampam/webhook-tester/internal/pkg/realip"
	"go.uber.org/zap"
)

// New creates mux.MiddlewareFunc for HTTP requests logging using "zap" package.
func New(log *zap.Logger) mux.MiddlewareFunc {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			metrics := httpsnoop.CaptureMetrics(next, w, r)

			log.Info("HTTP request processed",
				zap.String("remote addr", realip.FromHTTPRequest(r)),
				zap.String("useragent", r.UserAgent()),
				zap.String("method", r.Method),
				zap.String("url", r.URL.String()),
				zap.Int("status code", metrics.Code),
				zap.Duration("duration", metrics.Duration),
			)
		})
	}
}
