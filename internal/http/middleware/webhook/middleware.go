package webhook

import (
	"net/http"
	"strings"

	"github.com/google/uuid"
	"go.uber.org/zap"
)

const uuidLength = 36

func New(log *zap.Logger) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			// check if the request starts with a valid UUID
			if clean := strings.TrimLeft(r.URL.Path, "/"); len(clean) >= uuidLength && isValidUUID(clean[:uuidLength]) {
				_, _ = w.Write([]byte("WEBHOOK CAPTURED"))

				return
			}

			next.ServeHTTP(w, r)
		})
	}
}

// isValidUUID checks if passed string is valid UUID v4.
func isValidUUID(id string) bool {
	if len(id) != uuidLength {
		return false
	}

	_, err := uuid.Parse(id)

	return err == nil
}
