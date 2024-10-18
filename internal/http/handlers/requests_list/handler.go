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
	sID = openapi.SessionUUIDInPath

	Handler struct{ db storage.Storage }
)

func New(db storage.Storage) *Handler { return &Handler{db: db} }

func (h *Handler) Handle(ctx context.Context, sID sID) (*openapi.CapturedRequestsListResponse, error) {
	rList, lErr := h.db.GetAllRequests(ctx, sID.String())
	if lErr != nil {
		return nil, lErr
	}

	var list = make([]openapi.CapturedRequest, 0, len(rList))

	for rID, r := range rList {
		rUUID, pErr := uuid.Parse(rID)
		if pErr != nil {
			return nil, fmt.Errorf("failed to parse request UUID: %w", pErr)
		}

		var rHeaders = make([]openapi.HttpHeader, len(r.Headers))
		for i, header := range r.Headers {
			rHeaders[i].Name, rHeaders[i].Value = header.Name, header.Value
		}

		list = append(list, openapi.CapturedRequest{
			CapturedAtUnixMilli:  r.CreatedAt.UnixMilli(),
			ClientAddress:        r.ClientAddr,
			Headers:              rHeaders,
			Method:               strings.ToUpper(r.Method),
			RequestPayloadBase64: base64.StdEncoding.EncodeToString(r.Body),
			Url:                  r.URL,
			Uuid:                 rUUID,
		})

		// sort the list by the captured time from newest to oldest
		slices.SortFunc(list, func(a, b openapi.CapturedRequest) int {
			return int(b.CapturedAtUnixMilli - a.CapturedAtUnixMilli)
		})
	}

	return &list, nil
}
