package session_create

import (
	"context"
	"encoding/base64"
	"fmt"
	"time"

	"github.com/google/uuid"

	"gh.tarampamp.am/webhook-tester/v3/internal/httpserver/openapi"
	"gh.tarampamp.am/webhook-tester/v3/internal/storage"
)

// Handler handles HTTP requests for creating a new session.
type Handler struct {
	s   storage.SessionStorage
	ttl time.Duration
}

// New creates a new Handler.
func New(s storage.SessionStorage, ttl time.Duration) *Handler { return &Handler{s: s, ttl: ttl} }

// Handle creates a new session with the given parameters and returns its options.
func (h *Handler) Handle(
	ctx context.Context,
	req openapi.CreateSessionRequest,
) (*openapi.CreateSessionResponse, error) {
	body, err := base64.StdEncoding.DecodeString(req.ResponseBodyBase64)
	if err != nil {
		return nil, openapi.NewErrBadRequest("cannot decode response body: " + err.Error())
	}

	var headers []storage.ResponseHeader
	if len(req.Headers) > 0 {
		headers = make([]storage.ResponseHeader, len(req.Headers))
		for i, hdr := range req.Headers {
			headers[i] = storage.ResponseHeader(hdr)
		}
	}

	sID := uuid.New().String()

	meta, sErr := h.s.NewSession(ctx, sID, storage.SessionResponse{
		Code:    uint16(req.StatusCode), //nolint:gosec // validated: 200–530 fits in uint16
		Headers: headers,
		Body:    body,
		Delay:   time.Duration(req.Delay) * time.Second,
	}, h.ttl)
	if sErr != nil {
		return nil, fmt.Errorf("create session: %w", sErr)
	}

	respHeaders := make([]openapi.HttpHeader, len(req.Headers))
	for i, hdr := range req.Headers {
		respHeaders[i] = openapi.HttpHeader{Name: hdr.Name, Value: hdr.Value}
	}

	return &openapi.CreateSessionResponse{
		CreatedAtUnixMilli: meta.CreatedAt.UnixMilli(),
		Uuid:               sID,
	}, nil
}
