// Package version contains version handler.
package version

import (
	"net/http"

	"github.com/tarampampam/webhook-tester/internal/pkg/http/responder"

	jsoniter "github.com/json-iterator/go"
)

// NewHandler creates version handler.
func NewHandler(ver string) http.HandlerFunc {
	out := output{
		Version: ver,
	}

	return func(w http.ResponseWriter, _ *http.Request) {
		responder.JSON(w, out)
	}
}

type output struct {
	Version string `json:"version"`
}

func (o output) ToJSON() ([]byte, error) { return jsoniter.ConfigFastest.Marshal(o) }
