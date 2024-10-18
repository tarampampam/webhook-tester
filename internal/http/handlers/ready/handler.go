package ready

import (
	"context"
	"net/http"
)

type Handler struct{}

func New() *Handler { return &Handler{} }

func (h *Handler) HandleGet(context.Context, http.ResponseWriter)  {}
func (h *Handler) HandleHead(context.Context, http.ResponseWriter) {}
