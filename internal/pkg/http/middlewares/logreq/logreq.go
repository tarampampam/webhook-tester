// Package logreq contains middleware for HTTP requests logging using "zap" package.
package logreq

import (
	"net"
	"net/http"
	"strings"

	"github.com/felixge/httpsnoop"
	"github.com/gorilla/mux"
	"go.uber.org/zap"
)

// New creates mux.MiddlewareFunc for HTTP requests logging using "zap" package.
func New(log *zap.Logger) mux.MiddlewareFunc {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			metrics := httpsnoop.CaptureMetrics(next, w, r)

			log.Info("HTTP request processed",
				zap.String("remote_addr", getRealClientAddress(r)),
				zap.String("useragent", r.UserAgent()),
				zap.String("method", r.Method),
				zap.String("url", r.URL.String()),
				zap.Int("status_code", metrics.Code),
				zap.Int64("duration_micro", metrics.Duration.Microseconds()),
			)
		})
	}
}

// we will trust following HTTP headers for the real ip extracting (priority low -> high).
var trustHeaders = [...]string{"X-Forwarded-For", "X-Real-IP", "CF-Connecting-IP"} //nolint:gochecknoglobals

// getRealClientAddress extracts real client IP address from request.
func getRealClientAddress(r *http.Request) string {
	var ip string

	for _, name := range trustHeaders {
		if value := r.Header.Get(name); value != "" {
			// `X-Forwarded-For` can be `10.0.0.1, 10.0.0.2, 10.0.0.3`
			if strings.Contains(value, ",") {
				parts := strings.Split(value, ",")

				if len(parts) > 0 {
					ip = strings.TrimSpace(parts[0])
				}
			} else {
				ip = strings.TrimSpace(value)
			}
		}
	}

	if net.ParseIP(ip) != nil {
		return ip
	}

	return strings.Split(r.RemoteAddr, ":")[0]
}
