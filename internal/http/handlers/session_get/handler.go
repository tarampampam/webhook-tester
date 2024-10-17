package session_get

import "gh.tarampamp.am/webhook-tester/v2/internal/http/openapi"

type Handler struct{}

func New() *Handler { return &Handler{} }

func (h *Handler) Handle(openapi.SessionUUIDInPath) (*openapi.SessionOptionsResponse, error) {
	return &openapi.SessionOptionsResponse{}, nil
}
