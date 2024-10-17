package requests_subscribe

import (
	"net/http"

	"gh.tarampamp.am/webhook-tester/v2/internal/http/openapi"
)

type Handler struct{}

func New() *Handler { return &Handler{} }

func (h *Handler) Handle(http.ResponseWriter, *http.Request, openapi.SessionUUIDInPath) error {
	return nil
}
