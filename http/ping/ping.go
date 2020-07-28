package ping

import (
	"net/http"
)

type Handler struct{}

func NewHandler() http.Handler {
	return &Handler{}
}

func (h *Handler) ServeHTTP(w http.ResponseWriter, _ *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	_, _ = w.Write([]byte(`"pong"`))
}
