package session_get

import (
	"context"
	"encoding/base64"
	"fmt"

	"gh.tarampamp.am/webhook-tester/v2/internal/http/openapi"
	"gh.tarampamp.am/webhook-tester/v2/internal/storage"
)

type Handler struct{ db storage.Storage }

func New(db storage.Storage) *Handler { return &Handler{db: db} }

func (h *Handler) Handle(ctx context.Context, sID openapi.SessionUUIDInPath) (*openapi.SessionOptionsResponse, error) {
	sess, sErr := h.db.GetSession(ctx, sID.String())
	if sErr != nil {
		return nil, fmt.Errorf("failed to get session: %w", sErr)
	}

	var rHeaders = make(openapi.HttpHeaders, len(sess.Headers))
	for i, header := range sess.Headers {
		rHeaders[i].Name, rHeaders[i].Value = header.Name, header.Value
	}

	return &openapi.SessionOptionsResponse{
		CreatedAt: int(sess.CreatedAt.Unix()),
		Response: openapi.SessionResponseOptions{
			Delay:              uint16(sess.Delay.Seconds()),
			Headers:            rHeaders,
			ResponseBodyBase64: base64.StdEncoding.EncodeToString(sess.ResponseBody),
			StatusCode:         openapi.StatusCode(sess.Code),
		},
		Uuid: sID,
	}, nil
}
