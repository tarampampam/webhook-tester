package webhook

import (
	"net/http"
	"strings"

	"github.com/google/uuid"
	"go.uber.org/zap"
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

const uuidLength = 36

// shouldCaptureRequest checks if the request should be captured (the path starts with a valid UUID).
func shouldCaptureRequest(r *http.Request) bool {
	if r.URL == nil {
		return false
	}

	if clean := strings.TrimLeft(r.URL.Path, "/"); len(clean) >= uuidLength && isValidUUID(clean[:uuidLength]) {
		return true
	}

	return false
}

// isValidUUID checks if passed string is valid UUID v4.
func isValidUUID(id string) bool {
	if len(id) != uuidLength {
		return false
	}

	_, err := uuid.Parse(id)

	return err == nil
}
