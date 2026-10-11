package request_get

import (
	"context"
	"encoding/base64"
	"errors"
	"fmt"

	"gh.tarampamp.am/webhook-tester/v3/internal/httpserver/openapi"
	"gh.tarampamp.am/webhook-tester/v3/internal/storage"
)

type (
	sID = openapi.SessionUUIDInPath
	rID = openapi.RequestUUIDInPath
)

// Handler handles HTTP requests for retrieving a captured request.
type Handler struct {
	s storage.RequestStorage
}

// New creates a new Handler.
func New(s storage.RequestStorage) *Handler { return &Handler{s: s} }

// Handle retrieves the specified captured request from storage.
func (h *Handler) Handle(ctx context.Context, sessionID sID, requestID rID) (*openapi.CapturedRequestsResponse, error) {
	req, err := h.s.GetRequest(ctx, sessionID, requestID)
	if err != nil {
		switch {
		case errors.Is(err, storage.ErrSessionNotFound):
			return nil, openapi.NewErrNotFound("session not found")
		case errors.Is(err, storage.ErrRequestNotFound):
			return nil, openapi.NewErrNotFound("request not found")
		}

		return nil, fmt.Errorf("get request: %w", err)
	}

	headers := make([]openapi.HttpHeader, len(req.Data.Headers))
	for i, hdr := range req.Data.Headers {
		headers[i] = openapi.HttpHeader{Name: hdr.Name, Value: hdr.Value}
	}

	return &openapi.CapturedRequestsResponse{
		CapturedAtUnixMilli:  req.Meta.CreatedAt.UnixMilli(),
		ClientAddress:        req.Data.ClientAddr,
		Headers:              headers,
		Method:               req.Data.Method,
		RequestPayloadBase64: base64.StdEncoding.EncodeToString(req.Data.Body),
		Url:                  req.Data.URL,
		Uuid:                 requestID,
	}, nil
}
