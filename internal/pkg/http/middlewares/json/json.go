// Package json contains middleware for setting JSON content type header.
package json

import (
	"net/http"

	"github.com/gorilla/mux"
)

// New creates mux.MiddlewareFunc for setting JSON content type header.
func New() mux.MiddlewareFunc {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("Content-Type", "application/json")

			next.ServeHTTP(w, r)
		})
	}
}
