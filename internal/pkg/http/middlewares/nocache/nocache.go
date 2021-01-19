// Package nocache contains middleware for HTTP response caching disabling.
package nocache

import (
	"net/http"

	"github.com/gorilla/mux"
)

// New creates mux.MiddlewareFunc for HTTP response caching disabling.
func New() mux.MiddlewareFunc {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("Cache-Control", "no-cache, no-store, must-revalidate")
			w.Header().Set("Pragma", "no-cache")
			w.Header().Set("Expires", "0")

			next.ServeHTTP(w, r)
		})
	}
}
