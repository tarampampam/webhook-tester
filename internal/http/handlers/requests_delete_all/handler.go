package requests_delete_all

import "gh.tarampamp.am/webhook-tester/v2/internal/http/openapi"

type Handler struct{}

func New() *Handler { return &Handler{} }

func (h *Handler) Handle(openapi.SessionUUIDInPath) (*openapi.SuccessfulOperationResponse, error) {
	return &openapi.SuccessfulOperationResponse{}, nil
}
