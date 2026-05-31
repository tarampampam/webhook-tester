package requests_list

import (
	"context"
	"errors"

	"gh.tarampamp.am/webhook-tester/v3/internal/httpserver/openapi"
)

type sID = openapi.SessionUUIDInPath

// Handler handles HTTP requests for listing all captured requests of a session.
type Handler struct{}

// New creates a new Handler.
func New() *Handler { return &Handler{} }

// Handle returns all captured requests for the given session.
func (h *Handler) Handle(_ context.Context, _ sID) (*openapi.CapturedRequestsListResponse, error) {
	return nil, errors.New("not implemented")
}
