package requests_list

import (
	"context"
	"encoding/base64"
	"errors"
	"fmt"

	"gh.tarampamp.am/webhook-tester/v3/internal/httpserver/openapi"
	"gh.tarampamp.am/webhook-tester/v3/internal/storage"
)

type sID = openapi.SessionUUIDInPath

// Handler handles HTTP requests for listing all captured requests of a session.
type Handler struct {
	s storage.RequestStorage
}

// New creates a new Handler.
func New(s storage.RequestStorage) *Handler { return &Handler{s: s} }

// Handle returns all captured requests for the given session, ordered newest-first.
func (h *Handler) Handle(ctx context.Context, sessionID sID) (*openapi.CapturedRequestsListResponse, error) {
	var iterErr error

	seq, err := h.s.GetRequests(ctx, sessionID, &iterErr)
	if err != nil {
		if errors.Is(err, storage.ErrSessionNotFound) {
			return nil, openapi.NewErrNotFound("session not found")
		}

		return nil, fmt.Errorf("get requests: %w", err)
	}

	list := make(openapi.CapturedRequestsListResponse, 0)

	for rID, req := range seq {
		headers := make([]openapi.HttpHeader, len(req.Data.Headers))
		for i, hdr := range req.Data.Headers {
			headers[i] = openapi.HttpHeader{Name: hdr.Name, Value: hdr.Value}
		}

		list = append(list, openapi.CapturedRequest{
			CapturedAtUnixMilli:  req.Meta.CreatedAt.UnixMilli(),
			ClientAddress:        req.Data.ClientAddr,
			Headers:              headers,
			Method:               req.Data.Method,
			RequestPayloadBase64: base64.StdEncoding.EncodeToString(req.Data.Body),
			Url:                  req.Data.URL,
			Uuid:                 rID,
		})
	}

	if iterErr != nil {
		return nil, fmt.Errorf("iterate requests: %w", iterErr)
	}

	return &list, nil
}
