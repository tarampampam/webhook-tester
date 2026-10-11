package session_get

import (
	"context"
	"encoding/base64"
	"errors"
	"fmt"

	"gh.tarampamp.am/webhook-tester/v3/internal/httpserver/openapi"
	"gh.tarampamp.am/webhook-tester/v3/internal/storage"
)

type sID = openapi.SessionUUIDInPath

// Handler handles HTTP requests for retrieving session options.
type Handler struct {
	s storage.SessionStorage
}

// New creates a new Handler.
func New(s storage.SessionStorage) *Handler { return &Handler{s: s} }

// Handle returns the options for the specified session.
func (h *Handler) Handle(ctx context.Context, id sID) (*openapi.SessionOptionsResponse, error) {
	sess, err := h.s.GetSession(ctx, id)
	if err != nil {
		if errors.Is(err, storage.ErrSessionNotFound) {
			return nil, openapi.NewErrNotFound("session not found")
		}

		return nil, fmt.Errorf("get session: %w", err)
	}

	headers := make([]openapi.HttpHeader, len(sess.Response.Headers))
	for i, hdr := range sess.Response.Headers {
		headers[i] = openapi.HttpHeader{Name: hdr.Name, Value: hdr.Value}
	}

	return &openapi.SessionOptionsResponse{
		CreatedAtUnixMilli: sess.Meta.CreatedAt.UnixMilli(),
		Response: openapi.SessionResponseOptions{
			Delay:              uint16(sess.Response.Delay.Seconds()),
			Headers:            headers,
			ResponseBodyBase64: base64.StdEncoding.EncodeToString(sess.Response.Body),
			StatusCode:         int(sess.Response.Code),
		},
		Uuid: id,
	}, nil
}
