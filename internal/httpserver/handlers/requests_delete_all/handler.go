package requests_delete_all

import (
	"context"
	"errors"

	"gh.tarampamp.am/webhook-tester/v3/internal/httpserver/openapi"
)

type sID = openapi.SessionUUIDInPath

// Handler handles HTTP requests for deleting all captured requests of a session.
type Handler struct{}

// New creates a new Handler.
func New() *Handler { return &Handler{} }

// Handle deletes all requests for the specified session and notifies subscribers.
func (h *Handler) Handle(_ context.Context, _ sID) (*openapi.SuccessfulOperationResponse, error) {
	return nil, errors.New("not implemented")
}
