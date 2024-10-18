package cors

import "net/http"

// AllowAll creates a middleware that adds the CORS headers to the response that allows all origins, methods,
// and headers.
func AllowAll() func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("Access-Control-Allow-Origin", "*")
			w.Header().Set("Access-Control-Allow-Methods", "*")
			w.Header().Set("Access-Control-Allow-Headers", "*")

			next.ServeHTTP(w, r)
		})
	}
}
