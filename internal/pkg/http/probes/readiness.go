package probes

import (
	"net/http"

	"github.com/tarampampam/webhook-tester/internal/pkg/storage"
)

type Readiness struct {
	storage storage.Storage
}

func NewReadinessHandler(storage storage.Storage) http.Handler {
	return &Readiness{
		storage: storage,
	}
}

func (h *Readiness) ServeHTTP(w http.ResponseWriter, _ *http.Request) {
	if err := h.storage.Test(); err != nil {
		http.Error(w, err.Error(), http.StatusServiceUnavailable)
		return
	}

	w.WriteHeader(http.StatusOK)
}
