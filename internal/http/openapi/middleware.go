package openapi

import "net/http"

// CorsMiddleware is a middleware that adds the CORS headers to the response.
func CorsMiddleware() MiddlewareFunc {
	return func(h http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("Access-Control-Allow-Origin", "*") // TODO: limit the origin
			w.Header().Set("Access-Control-Allow-Methods", "*")
			w.Header().Set("Access-Control-Allow-Headers", "*")

			h.ServeHTTP(w, r)
		})
	}
}
