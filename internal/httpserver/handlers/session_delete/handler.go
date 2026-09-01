package session_delete

import (
	"context"
	"errors"
	"fmt"

	"gh.tarampamp.am/webhook-tester/v3/internal/httpserver/openapi"
	"gh.tarampamp.am/webhook-tester/v3/internal/storage"
)

type sID = openapi.SessionUUIDInPath

// Handler handles HTTP requests for deleting a session.
type Handler struct {
	s storage.SessionStorage
}

// New creates a new Handler.
func New(s storage.SessionStorage) *Handler { return &Handler{s: s} }

// Handle deletes the specified session from storage.
func (h *Handler) Handle(ctx context.Context, id sID) (*openapi.SuccessfulOperationResponse, error) {
	if err := h.s.DeleteSession(ctx, id); err != nil {
		if errors.Is(err, storage.ErrSessionNotFound) {
			return nil, openapi.NewErrNotFound("session not found")
		}

		return nil, fmt.Errorf("delete session: %w", err)
	}

	return &openapi.SuccessfulOperationResponse{Success: true}, nil
}
