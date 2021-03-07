// Package version contains version API handler.
package version

import (
	"net/http"

	jsoniter "github.com/json-iterator/go"
)

// NewHandler creates version handler.
func NewHandler(ver string) http.HandlerFunc {
	var cache []byte

	return func(w http.ResponseWriter, _ *http.Request) {
		if cache == nil {
			cache, _ = jsoniter.ConfigFastest.Marshal(struct {
				Version string `json:"version"`
			}{
				Version: ver,
			})
		}

		w.WriteHeader(http.StatusOK)
		_, _ = w.Write(cache)
	}
}
