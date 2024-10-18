package ready

import (
	"context"
	"net/http"
)

type Handler struct{}

func New() *Handler { return &Handler{} }

func (h *Handler) HandleGet(_ context.Context, w http.ResponseWriter) { _, _ = w.Write([]byte("OK")) }
func (h *Handler) HandleHead(context.Context, http.ResponseWriter)    {}
