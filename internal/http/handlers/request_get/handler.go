package request_get

import (
	"context"
	"encoding/base64"
	"strings"

	"gh.tarampamp.am/webhook-tester/v2/internal/http/openapi"
	"gh.tarampamp.am/webhook-tester/v2/internal/storage"
)

type (
	sID = openapi.SessionUUIDInPath
	rID = openapi.RequestUUIDInPath

	Handler struct{ db storage.Storage }
)

func New(db storage.Storage) *Handler { return &Handler{db: db} }

func (h *Handler) Handle(ctx context.Context, sID sID, rID rID) (*openapi.CapturedRequestsResponse, error) {
	r, rErr := h.db.GetRequest(ctx, sID.String(), rID.String())
	if rErr != nil {
		return nil, rErr
	}

	var rHeaders = make([]openapi.HttpHeader, len(r.Headers))
	for i, header := range r.Headers {
		rHeaders[i].Name, rHeaders[i].Value = header.Name, header.Value
	}

	return &openapi.CapturedRequestsResponse{
		CapturedAtUnixMilli:  r.CreatedAtUnixMilli,
		ClientAddress:        r.ClientAddr,
		Headers:              rHeaders,
		Method:               strings.ToUpper(r.Method),
		RequestPayloadBase64: base64.StdEncoding.EncodeToString(r.Body),
		Url:                  r.URL,
		Uuid:                 rID,
	}, nil
}
