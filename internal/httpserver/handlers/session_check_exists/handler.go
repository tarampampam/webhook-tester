package session_check_exists

import (
	"context"
	"errors"

	"gh.tarampamp.am/webhook-tester/v3/internal/httpserver/openapi"
)

// Handler handles HTTP requests for checking whether sessions exist.
type Handler struct{}

// New creates a new Handler.
func New() *Handler { return &Handler{} }

// Handle checks which of the given session IDs exist and returns a map of ID to existence flag.
func (h *Handler) Handle(_ context.Context, _ []openapi.UUID) (*openapi.CheckSessionExistsResponse, error) {
	return nil, errors.New("not implemented")
}
