package live

import "net/http"

type Handler struct{}

func New() *Handler { return &Handler{} }

func (h *Handler) Handle(w http.ResponseWriter, method string) {
	w.Header().Set("Content-Type", "text/plain; charset=utf-8")
	w.WriteHeader(http.StatusOK)

	if method == http.MethodGet {
		_, _ = w.Write([]byte("OK"))
	}
}
