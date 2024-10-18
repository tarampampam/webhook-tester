package session_create

import (
	"context"

	"gh.tarampamp.am/webhook-tester/v2/internal/http/openapi"
)

type Handler struct{}

func New() *Handler { return &Handler{} }

func (h *Handler) Handle(context.Context, openapi.CreateSessionRequest) (*openapi.SessionOptionsResponse, error) {
	return &openapi.SessionOptionsResponse{}, nil
}
