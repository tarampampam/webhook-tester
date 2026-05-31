package request_delete

import (
	"context"
	"errors"

	"gh.tarampamp.am/webhook-tester/v3/internal/httpserver/openapi"
)

type (
	sID = openapi.SessionUUIDInPath
	rID = openapi.RequestUUIDInPath
)

// Handler handles HTTP requests for deleting a captured request.
type Handler struct{}

// New creates a new Handler.
func New() *Handler { return &Handler{} }

// Handle deletes the specified request from storage and notifies subscribers.
func (h *Handler) Handle(_ context.Context, _ sID, _ rID) (*openapi.SuccessfulOperationResponse, error) {
	return nil, errors.New("not implemented")
}
