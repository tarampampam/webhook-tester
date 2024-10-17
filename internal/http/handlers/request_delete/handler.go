package request_delete

import "gh.tarampamp.am/webhook-tester/v2/internal/http/openapi"

type (
	sID = openapi.SessionUUIDInPath
	rID = openapi.RequestUUIDInPath

	Handler struct{}
)

func New() *Handler { return &Handler{} }

func (h *Handler) Handle(sID, rID) (*openapi.SuccessfulOperationResponse, error) {
	return &openapi.SuccessfulOperationResponse{}, nil
}
