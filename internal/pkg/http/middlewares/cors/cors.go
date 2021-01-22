// Package cors contains middleware for cross-original requests allowing.
package cors

import (
	"net/http"

	"github.com/gorilla/mux"
)

// New creates mux.MiddlewareFunc for cross-original requests allowing.
func New() mux.MiddlewareFunc {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("Access-Control-Allow-Origin", "*")

			next.ServeHTTP(w, r)
		})
	}
}
