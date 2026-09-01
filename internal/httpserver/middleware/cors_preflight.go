package middleware

import "net/http"

// CORSPreflight returns a middleware that silently handles browser CORS preflight requests.
//
// Browser preflights are identified by the Access-Control-Request-Method header, which the browser includes
// only in automatically generated OPTIONS preflight requests - never in user-initiated OPTIONS requests
// (e.g., fetch with method: "OPTIONS"). When this fingerprint is detected, the middleware responds with
// 204 No Content and the appropriate CORS headers without forwarding to the next handler.
//
// Access-Control-Allow-Headers mirrors the request's Access-Control-Request-Headers value, which covers
// all headers the browser intends to send including Authorization (not covered by the * wildcard).
func CORSPreflight(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodOptions && r.Header.Get("Access-Control-Request-Method") != "" {
			hdr := w.Header()
			hdr.Set("Access-Control-Allow-Origin", "*")
			hdr.Set("Access-Control-Allow-Methods", r.Header.Get("Access-Control-Request-Method"))

			if reqHeaders := r.Header.Get("Access-Control-Request-Headers"); reqHeaders != "" {
				hdr.Set("Access-Control-Allow-Headers", reqHeaders)
			}

			hdr.Set("Access-Control-Max-Age", "86400")
			w.WriteHeader(http.StatusNoContent)

			return
		}

		next.ServeHTTP(w, r)
	})
}
