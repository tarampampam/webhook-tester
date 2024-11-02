package requests_delete_all

import (
	"context"

	"gh.tarampamp.am/webhook-tester/v2/internal/http/openapi"
	"gh.tarampamp.am/webhook-tester/v2/internal/storage"
)

type (
	sID = openapi.SessionUUIDInPath

	Handler struct{ db storage.Storage }
)

func New(db storage.Storage) *Handler { return &Handler{db: db} }

func (h *Handler) Handle(ctx context.Context, sID sID) (*openapi.SuccessfulOperationResponse, error) {
	if err := h.db.DeleteAllRequests(ctx, sID.String()); err != nil {
		return nil, err
	}

	return &openapi.SuccessfulOperationResponse{Success: true}, nil
}
