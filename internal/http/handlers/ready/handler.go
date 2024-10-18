package ready

import (
	"context"
	"net/http"
)

type (
	checker func(context.Context) error
	Handler struct{ checker checker }
)

func New(c checker) *Handler { return &Handler{checker: c} }

func (h *Handler) Handle(ctx context.Context, w http.ResponseWriter, method string) {
	w.Header().Set("Content-Type", "text/plain; charset=utf-8")

	if h.checker == nil {
		w.WriteHeader(http.StatusNoContent)

		return
	}

	if err := h.checker(ctx); err != nil {
		w.WriteHeader(http.StatusServiceUnavailable)

		if method == http.MethodGet {
			_, _ = w.Write([]byte(err.Error()))
		}

		return
	}

	w.WriteHeader(http.StatusOK)

	if method == http.MethodGet {
		_, _ = w.Write([]byte("OK"))
	}
}
