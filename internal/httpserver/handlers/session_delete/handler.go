package session_delete

import (
	"context"
	"errors"

	"gh.tarampamp.am/webhook-tester/v3/internal/httpserver/openapi"
)

type sID = openapi.SessionUUIDInPath

// Handler handles HTTP requests for deleting a session.
type Handler struct{}

// New creates a new Handler.
func New() *Handler { return &Handler{} }

// Handle deletes the specified session from storage.
func (h *Handler) Handle(_ context.Context, _ sID) (*openapi.SuccessfulOperationResponse, error) {
	return nil, errors.New("not implemented")
}
