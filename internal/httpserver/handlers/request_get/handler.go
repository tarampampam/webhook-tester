package request_get

import (
	"context"
	"errors"

	"gh.tarampamp.am/webhook-tester/v3/internal/httpserver/openapi"
)

type (
	sID = openapi.SessionUUIDInPath
	rID = openapi.RequestUUIDInPath
)

// Handler handles HTTP requests for retrieving a captured request.
type Handler struct{}

// New creates a new Handler.
func New() *Handler { return &Handler{} }

// Handle retrieves the specified captured request from storage.
func (h *Handler) Handle(_ context.Context, _ sID, _ rID) (*openapi.CapturedRequestsResponse, error) {
	return nil, errors.New("not implemented")
}
