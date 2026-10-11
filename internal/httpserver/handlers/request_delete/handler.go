package request_delete

import (
	"context"
	"errors"
	"fmt"

	"gh.tarampamp.am/webhook-tester/v3/internal/httpserver/openapi"
	"gh.tarampamp.am/webhook-tester/v3/internal/pubsub"
	"gh.tarampamp.am/webhook-tester/v3/internal/storage"
)

type (
	sID = openapi.SessionUUIDInPath
	rID = openapi.RequestUUIDInPath
)

// Handler handles HTTP requests for deleting a captured request.
type Handler struct {
	s  storage.RequestStorage
	ps pubsub.Publisher
}

// New creates a new Handler.
func New(s storage.RequestStorage, ps pubsub.Publisher) *Handler { return &Handler{s: s, ps: ps} }

// Handle deletes the specified request from storage and notifies subscribers.
func (h *Handler) Handle(
	ctx context.Context, sessionID sID, requestID rID,
) (*openapi.SuccessfulOperationResponse, error) {
	if err := h.s.DeleteRequest(ctx, sessionID, requestID); err != nil {
		switch {
		case errors.Is(err, storage.ErrSessionNotFound):
			return nil, openapi.NewErrNotFound("session not found")
		case errors.Is(err, storage.ErrRequestNotFound):
			return nil, openapi.NewErrNotFound("request not found")
		}

		return nil, fmt.Errorf("delete request: %w", err)
	}

	if err := h.ps.Publish(ctx, sessionID, pubsub.RequestEvent{
		Action:  pubsub.RequestActionDelete,
		Request: &pubsub.RequestData{ID: requestID},
	}); err != nil {
		return nil, fmt.Errorf("publish event: %w", err)
	}

	return &openapi.SuccessfulOperationResponse{Success: true}, nil
}
