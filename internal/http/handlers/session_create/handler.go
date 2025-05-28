package session_create

import (
	"context"
	"encoding/base64"
	"fmt"
	"time"

	"github.com/google/uuid"

	"gh.tarampamp.am/webhook-tester/v2/internal/http/openapi"
	"gh.tarampamp.am/webhook-tester/v2/internal/storage"
)

type Handler struct{ db storage.Storage }

func New(db storage.Storage) *Handler { return &Handler{db: db} }

func (h *Handler) Handle(ctx context.Context, p openapi.CreateSessionRequest) (*openapi.SessionOptionsResponse, error) {
	var sHeaders = make([]storage.HttpHeader, len(p.Headers))
	for i, header := range p.Headers {
		sHeaders[i] = storage.HttpHeader{Name: header.Name, Value: header.Value}
	}

	var responseBody, decErr = base64.StdEncoding.DecodeString(p.ResponseBodyBase64)
	if decErr != nil {
		return nil, fmt.Errorf("cannot decode response body (wrong base64): %w", decErr)
	}

	// Handle proxy configuration
	var proxyURLs []string
	if p.ProxyUrls != nil {
		proxyURLs = *p.ProxyUrls
	}

	var proxyResponseMode = "app_response" // Default value
	if p.ProxyResponseMode != nil && *p.ProxyResponseMode != "" {
		proxyResponseMode = *p.ProxyResponseMode
		// TODO: Add validation for proxyResponseMode if specific values are expected, e.g., "app_response", "proxy_first_success"
	}

	sID, sErr := h.db.NewSession(ctx, storage.Session{
		Code:              uint16(p.StatusCode), //nolint:gosec
		Headers:           sHeaders,
		ResponseBody:      responseBody,
		Delay:             time.Second * time.Duration(p.Delay),
		ProxyURLs:         proxyURLs,
		ProxyResponseMode: proxyResponseMode,
	})
	if sErr != nil {
		return nil, fmt.Errorf("failed to create a new session: %w", sErr)
	}

	sess, sErr := h.db.GetSession(ctx, sID)
	if sErr != nil {
		return nil, fmt.Errorf("failed to get session: %w", sErr)
	}

	sUUID, pErr := uuid.Parse(sID)
	if pErr != nil {
		return nil, fmt.Errorf("failed to parse session UUID: %w", pErr)
	}

	var rHeaders = make([]openapi.HttpHeader, len(sess.Headers))
	for i, header := range sess.Headers {
		rHeaders[i].Name, rHeaders[i].Value = header.Name, header.Value
	}

	// Prepare response, including proxy configuration
	var responseProxyUrls *[]string
	if len(sess.ProxyURLs) > 0 {
		responseProxyUrls = &sess.ProxyURLs
	}

	var responseProxyResponseMode *string
	if sess.ProxyResponseMode != "" {
		responseProxyResponseMode = &sess.ProxyResponseMode
	}

	return &openapi.SessionOptionsResponse{
		CreatedAtUnixMilli: sess.CreatedAtUnixMilli,
		Response: openapi.SessionResponseOptions{
			Delay:              uint16(sess.Delay.Seconds()),
			Headers:            rHeaders,
			ResponseBodyBase64: base64.StdEncoding.EncodeToString(sess.ResponseBody),
			StatusCode:         openapi.StatusCode(sess.Code),
			ProxyUrls:          responseProxyUrls,
			ProxyResponseMode:  responseProxyResponseMode,
		},
		Uuid: sUUID,
	}, nil
}
