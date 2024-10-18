package webhook

import (
	"net/http"
	"strings"

	"go.uber.org/zap"

	"gh.tarampamp.am/webhook-tester/v2/internal/http/openapi"
)

func New(log *zap.Logger) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if shouldCaptureRequest(r) {
				// set the header to allow CORS requests from any origin and method
				w.Header().Set("Access-Control-Allow-Origin", "*")
				w.Header().Set("Access-Control-Allow-Methods", "*")
				w.Header().Set("Access-Control-Allow-Headers", "*")

				_, _ = w.Write([]byte("WEBHOOK CAPTURED"))

				return
			}

			next.ServeHTTP(w, r)
		})
	}
}

// shouldCaptureRequest checks if the request should be captured (the path starts with a valid UUID).
func shouldCaptureRequest(r *http.Request) bool {
	if r.URL == nil {
		return false
	}

	var clean = strings.TrimLeft(r.URL.Path, "/")

	if len(clean) >= openapi.UUIDLength && openapi.IsValidUUID(clean[:openapi.UUIDLength]) {
		return true
	}

	return false
}
