package requests_delete_all

import (
	"context"
	"errors"
	"fmt"

	"gh.tarampamp.am/webhook-tester/v3/internal/httpserver/openapi"
	"gh.tarampamp.am/webhook-tester/v3/internal/pubsub"
	"gh.tarampamp.am/webhook-tester/v3/internal/storage"
)

type sID = openapi.SessionUUIDInPath

// Handler handles HTTP requests for deleting all captured requests of a session.
type Handler struct {
	s  storage.RequestStorage
	ps pubsub.Publisher
}

// New creates a new Handler.
func New(s storage.RequestStorage, ps pubsub.Publisher) *Handler { return &Handler{s: s, ps: ps} }

// Handle deletes all requests for the specified session and notifies subscribers.
func (h *Handler) Handle(ctx context.Context, sessionID sID) (*openapi.SuccessfulOperationResponse, error) {
	if err := h.s.DeleteAllRequests(ctx, sessionID); err != nil {
		if errors.Is(err, storage.ErrSessionNotFound) {
			return nil, openapi.NewErrNotFound("session not found")
		}

		return nil, fmt.Errorf("delete all requests: %w", err)
	}

	if err := h.ps.Publish(ctx, sessionID, pubsub.RequestEvent{Action: pubsub.RequestActionClear}); err != nil {
		return nil, fmt.Errorf("publish event: %w", err)
	}

	return &openapi.SuccessfulOperationResponse{Success: true}, nil
}
