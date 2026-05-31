package session_create

import (
	"context"
	"errors"

	"gh.tarampamp.am/webhook-tester/v3/internal/httpserver/openapi"
)

// Handler handles HTTP requests for creating a new session.
type Handler struct{}

// New creates a new Handler.
func New() *Handler { return &Handler{} }

// Handle creates a new session with the given parameters and returns its options.
func (h *Handler) Handle(_ context.Context, _ openapi.CreateSessionRequest) (*openapi.SessionOptionsResponse, error) {
	return nil, errors.New("not implemented")
}
