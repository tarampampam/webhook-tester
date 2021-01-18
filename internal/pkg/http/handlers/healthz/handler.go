// Package healthz contains healthcheck handler.
package healthz

import (
	"net/http"
)

// checker allows to check some service part.
type checker interface {
	// Check makes a check and return error only if something is wrong.
	Check() error
}

// NewHandler creates healthcheck handler.
func NewHandler(checker checker) http.HandlerFunc {
	return func(w http.ResponseWriter, _ *http.Request) {
		if err := checker.Check(); err != nil {
			w.WriteHeader(http.StatusServiceUnavailable)
			_, _ = w.Write([]byte(err.Error()))

			return
		}

		w.WriteHeader(http.StatusOK)
	}
}
