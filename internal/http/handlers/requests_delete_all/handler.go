package requests_delete_all

import (
	"context"

	"gh.tarampamp.am/webhook-tester/v2/internal/http/openapi"
	"gh.tarampamp.am/webhook-tester/v2/internal/pubsub"
	"gh.tarampamp.am/webhook-tester/v2/internal/storage"
)

type (
	sID = openapi.SessionUUIDInPath

	Handler struct {
		appCtx context.Context
		db     storage.Storage
		pub    pubsub.Publisher[pubsub.RequestEvent]
	}
)

func New(appCtx context.Context, db storage.Storage, pub pubsub.Publisher[pubsub.RequestEvent]) *Handler {
	return &Handler{appCtx: appCtx, db: db, pub: pub}
}

func (h *Handler) Handle(ctx context.Context, sID sID) (*openapi.SuccessfulOperationResponse, error) {
	if err := h.db.DeleteAllRequests(ctx, sID.String()); err != nil {
		return nil, err
	}

	// notify the subscribers
	if err := h.pub.Publish(h.appCtx, sID.String(), pubsub.RequestEvent{Action: pubsub.RequestActionClear}); err != nil { //nolint:contextcheck,lll
		return nil, err
	}

	return &openapi.SuccessfulOperationResponse{Success: true}, nil
}
