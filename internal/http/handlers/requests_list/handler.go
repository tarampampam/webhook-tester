package requests_list

import (
	"context"

	"gh.tarampamp.am/webhook-tester/v2/internal/http/openapi"
)

type Handler struct{}

func New() *Handler { return &Handler{} }

func (h *Handler) Handle(context.Context, openapi.SessionUUIDInPath) (*openapi.CapturedRequestsListResponse, error) {
	return &openapi.CapturedRequestsListResponse{}, nil
}
