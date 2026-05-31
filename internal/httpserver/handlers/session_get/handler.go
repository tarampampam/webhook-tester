package session_get

import (
	"context"
	"errors"

	"gh.tarampamp.am/webhook-tester/v3/internal/httpserver/openapi"
)

type sID = openapi.SessionUUIDInPath

// Handler handles HTTP requests for retrieving session options.
type Handler struct{}

// New creates a new Handler.
func New() *Handler { return &Handler{} }

// Handle returns the options for the specified session.
func (h *Handler) Handle(_ context.Context, _ sID) (*openapi.SessionOptionsResponse, error) {
	return nil, errors.New("not implemented")
}
