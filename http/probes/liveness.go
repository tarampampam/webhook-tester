package probes

import "net/http"

type Liveness struct{}

func NewLivenessHandler() http.Handler {
	return &Liveness{}
}

func (h *Liveness) ServeHTTP(w http.ResponseWriter, _ *http.Request) {
	w.WriteHeader(http.StatusOK)
}
