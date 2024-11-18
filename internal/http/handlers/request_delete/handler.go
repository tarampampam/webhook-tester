package request_delete

import (
	"context"

	"gh.tarampamp.am/webhook-tester/v2/internal/http/openapi"
	"gh.tarampamp.am/webhook-tester/v2/internal/pubsub"
	"gh.tarampamp.am/webhook-tester/v2/internal/storage"
)

type (
	sID = openapi.SessionUUIDInPath
	rID = openapi.RequestUUIDInPath

	Handler struct {
		appCtx context.Context
		db     storage.Storage
		pub    pubsub.Publisher[pubsub.RequestEvent]
	}
)

func New(appCtx context.Context, db storage.Storage, pub pubsub.Publisher[pubsub.RequestEvent]) *Handler {
	return &Handler{appCtx: appCtx, db: db, pub: pub}
}

func (h *Handler) Handle(ctx context.Context, sID sID, rID rID) (*openapi.SuccessfulOperationResponse, error) {
	// get the request from the storage to notify the subscribers
	req, getErr := h.db.GetRequest(ctx, sID.String(), rID.String())
	if getErr != nil {
		return nil, getErr
	}

	// delete it
	if err := h.db.DeleteRequest(ctx, sID.String(), rID.String()); err != nil {
		return nil, err
	}

	// convert headers to the pubsub format
	var headers = make([]pubsub.HttpHeader, len(req.Headers))
	for i, rh := range req.Headers {
		headers[i] = pubsub.HttpHeader{Name: rh.Name, Value: rh.Value}
	}

	// notify the subscribers
	if err := h.pub.Publish(h.appCtx, sID.String(), pubsub.RequestEvent{ //nolint:contextcheck
		Action: pubsub.RequestActionDelete,
		Request: &pubsub.Request{
			ID:                 rID.String(),
			ClientAddr:         req.ClientAddr,
			Method:             req.Method,
			Headers:            headers,
			URL:                req.URL,
			CreatedAtUnixMilli: req.CreatedAtUnixMilli,
		},
	}); err != nil {
		return nil, err
	}

	return &openapi.SuccessfulOperationResponse{Success: true}, nil
}
