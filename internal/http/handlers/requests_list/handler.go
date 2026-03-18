package requests_list

import (
	"context"
	"encoding/base64"
	"fmt"
	"slices"
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
	rList, lErr := h.db.GetAllRequests(ctx, sID.String())
	if lErr != nil {
		return nil, lErr
	}

	var (
		includeBody = p.IncludeBody == nil || *p.IncludeBody
		list        = make([]openapi.CapturedRequest, 0, len(rList))
	)

	for rID, r := range rList {
		rUUID, pErr := uuid.Parse(rID)
		if pErr != nil {
			return nil, fmt.Errorf("failed to parse request UUID: %w", pErr)
		}

		var rHeaders = make([]openapi.HttpHeader, len(r.Headers))
		for i, header := range r.Headers {
			rHeaders[i].Name, rHeaders[i].Value = header.Name, header.Value
		}

		var payload string
		if includeBody {
			payload = base64.StdEncoding.EncodeToString(r.Body)
		}

		list = append(list, openapi.CapturedRequest{
			CapturedAtUnixMilli:  r.CreatedAtUnixMilli,
			ClientAddress:        r.ClientAddr,
			Headers:              rHeaders,
			Method:               strings.ToUpper(r.Method),
			RequestPayloadBase64: payload,
			Url:                  r.URL,
			Uuid:                 rUUID,
		})
	}

	// sort the list by the captured time from newest to oldest
	slices.SortFunc(list, func(a, b openapi.CapturedRequest) int {
		return int(b.CapturedAtUnixMilli - a.CapturedAtUnixMilli)
	})

	// apply offset
	if p.Offset != nil {
		if int(*p.Offset) < len(list) {
			list = list[*p.Offset:]
		} else {
			list = list[:0]
		}
	}

	// apply limit
	if p.Limit != nil && *p.Limit > 0 && int(*p.Limit) < len(list) {
		list = list[:*p.Limit]
	}

	return &list, nil
}
