package request_delete

import (
	"context"

	"gh.tarampamp.am/webhook-tester/v2/internal/http/openapi"
	"gh.tarampamp.am/webhook-tester/v2/internal/storage"
)

type (
	sID = openapi.SessionUUIDInPath
	rID = openapi.RequestUUIDInPath

	Handler struct{ db storage.Storage }
)

func New(db storage.Storage) *Handler { return &Handler{db: db} }

func (h *Handler) Handle(ctx context.Context, sID sID, rID rID) (*openapi.SuccessfulOperationResponse, error) {
	if err := h.db.DeleteRequest(ctx, sID.String(), rID.String()); err != nil {
		return nil, err
	}

	return &openapi.SuccessfulOperationResponse{Success: true}, nil
}
