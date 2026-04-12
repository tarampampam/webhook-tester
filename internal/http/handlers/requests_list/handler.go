package requests_list

import (
	"context"
	"encoding/base64"
	"fmt"
	"strings"

	"github.com/google/uuid"

	"gh.tarampamp.am/webhook-tester/v2/internal/http/openapi"
	"gh.tarampamp.am/webhook-tester/v2/internal/storage"
)

type (
	sID    = openapi.SessionUUIDInPath
	params = openapi.ApiSessionListRequestsParams

	Handler struct{ db storage.Storage }
)

func New(db storage.Storage) *Handler { return &Handler{db: db} }

func (h *Handler) Handle(ctx context.Context, sID sID, p params) (*openapi.CapturedRequestsListResponse, error) {
	var opts = storage.GetRequestsOptions{
		IncludeBody: p.IncludeBody == nil || *p.IncludeBody,
	}

	if p.Limit != nil {
		opts.Limit = *p.Limit
	}

	if p.Offset != nil {
		opts.Offset = *p.Offset
	}

	// storage returns already sorted (newest-first) and paginated results
	requests, err := h.db.GetRequests(ctx, sID.String(), opts)
	if err != nil {
		return nil, err
	}

	var list = make([]openapi.CapturedRequest, 0, len(requests))

	for _, r := range requests {
		rUUID, pErr := uuid.Parse(r.ID)
		if pErr != nil {
			return nil, fmt.Errorf("failed to parse request UUID: %w", pErr)
		}

		var rHeaders = make([]openapi.HttpHeader, len(r.Request.Headers))
		for i, header := range r.Request.Headers {
			rHeaders[i].Name, rHeaders[i].Value = header.Name, header.Value
		}

		list = append(list, openapi.CapturedRequest{
			CapturedAtUnixMilli:  r.Request.CreatedAtUnixMilli,
			ClientAddress:        r.Request.ClientAddr,
			Headers:              rHeaders,
			Method:               strings.ToUpper(r.Request.Method),
			RequestPayloadBase64: base64.StdEncoding.EncodeToString(r.Request.Body),
			Url:                  r.Request.URL,
			Uuid:                 rUUID,
		})
	}

	return &list, nil
}
