package live

import "net/http"

type Handler struct{}

func New() *Handler { return &Handler{} }

func (h *Handler) HandleGet(http.ResponseWriter)  { return }
func (h *Handler) HandleHead(http.ResponseWriter) { return }
