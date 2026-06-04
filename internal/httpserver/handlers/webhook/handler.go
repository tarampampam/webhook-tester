package webhook

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"net/http"
	"path"
	"slices"
	"strconv"
	"strings"
	"time"

	"github.com/google/uuid"

	"gh.tarampamp.am/webhook-tester/v3/internal/httpserver/middleware"
	"gh.tarampamp.am/webhook-tester/v3/internal/logger"
	"gh.tarampamp.am/webhook-tester/v3/internal/pubsub"
	"gh.tarampamp.am/webhook-tester/v3/internal/storage"
)

// ShouldBeCaptured checks if the incoming request should be skipped by the webhook handler, or not.
//
// It returns true for the following patterns:
//   - /{uuid} (e.g. /123e4567-e89b-12d3-a456-426614174000)
//   - /{uuid}/ (e.g. /123e4567-e89b-12d3-a456-426614174000/)
//   - /{uuid}/{anything} (e.g. /123e4567-e89b-12d3-a456-426614174000/fooBar_baz-123)
//
// Otherwise, it returns false, meaning that the request should NOT be handled by the webhook handler.
func ShouldBeCaptured(r *http.Request) bool { _, ok := extractSessionID(r); return ok } //nolint:nlreturn

// Handler is the HTTP handler for capturing webhook requests.
type Handler struct {
	log *logger.Logger
	s   storage.Storage
	ps  pubsub.Publisher

	autoCreateSessions bool
	sessionTTL         time.Duration
	maxRequestBodySize int64
}

// New creates a new Handler for capturing webhook requests.
func New(
	log *logger.Logger,
	s storage.Storage,
	ps pubsub.Publisher,
	autoCreateSessions bool,
	sessionTTL time.Duration,
	maxRequestBodySize uint,
) *Handler {
	return &Handler{
		log:                log,
		s:                  s,
		ps:                 ps,
		autoCreateSessions: autoCreateSessions,
		sessionTTL:         sessionTTL,
		maxRequestBodySize: int64(maxRequestBodySize), //nolint:gosec
	}
}

// Handle processes the incoming HTTP request, captures it, stores it in the storage, and publishes an event about
// the captured request.
func (h *Handler) Handle(w http.ResponseWriter, r *http.Request) { //nolint:funlen
	sID, sidOk := extractSessionID(r)
	if !sidOk {
		renderErrorMessage(h.log, w, r, http.StatusBadRequest, "invalid session ID in URL")

		return
	}

	// get (or create, if enabled) the session from the storage
	s, sEss := h.s.GetSession(r.Context(), sID)
	if sEss != nil { //nolint:nestif
		if errors.Is(sEss, storage.ErrNotFound) {
			if h.autoCreateSessions {
				if _, err := h.s.NewSession(r.Context(), sID, storage.SessionResponse{
					Code: http.StatusOK,
				}, h.sessionTTL); err != nil {
					renderError(h.log, w, r, err)

					return
				}

				s, sEss = h.s.GetSession(r.Context(), sID)
				if sEss != nil {
					renderError(h.log, w, r, sEss)

					return
				}

				// add the header indicating that the session has been created automatically
				w.Header().Set("X-Wh-Created-Automatically", "1")
			} else {
				renderErrorMessage(h.log, w, r,
					http.StatusNotFound,
					"The webhook hasn't been created yet, or it may have expired",
				)

				return
			}
		} else {
			renderError(h.log, w, r, sEss)

			return
		}
	}

	// refresh session TTL, if it has expiration
	if !s.Meta.ExpiresAt.IsZero() {
		if err := h.s.AddSessionTTL(r.Context(), sID, h.sessionTTL); err != nil {
			renderError(h.log, w, r, err)

			return
		}
	}

	var body bytes.Buffer

	if r.Body == nil {
		r.Body = http.NoBody
	}

	// read the request body, applying a size cap only when a limit is configured (0 means unlimited)
	var limitedBody io.Reader = r.Body
	if h.maxRequestBodySize > 0 {
		limitedBody = io.LimitReader(r.Body, h.maxRequestBodySize+1)
	}

	n, rErr := body.ReadFrom(limitedBody)
	if rErr != nil {
		renderErrorMessage(h.log, w, r,
			http.StatusInternalServerError,
			"Failed to read request body: "+rErr.Error(),
		)

		return
	}

	if h.maxRequestBodySize > 0 && n > h.maxRequestBodySize {
		renderErrorMessage(h.log, w, r, http.StatusRequestEntityTooLarge, "Request body is too large")

		return
	}

	// collect and sort the request headers for consistent storage and later retrieval
	headers := make([]storage.RequestHeader, 0, len(r.Header))
	for name, value := range r.Header {
		headers = append(headers, storage.RequestHeader{Name: name, Value: strings.Join(value, "; ")})
	}

	slices.SortFunc(headers, func(i, j storage.RequestHeader) int { return strings.Compare(i.Name, j.Name) })

	// generate new, unique request ID for the captured request
	rID := uuid.New().String()

	// try to get the original URL from the context (if the middleware is applied), otherwise, extract the full
	// URL manually
	rUrl := func(r *http.Request) string {
		if u, ok := middleware.OriginalURLFromContext(r.Context()); ok {
			return u.String()
		}

		return extractFullUrl(r)
	}(r)

	captured := storage.CapturedRequest{
		ClientAddr: extractRealIP(r),
		Method:     r.Method,
		URL:        rUrl,
		Headers:    headers,
		Body:       body.Bytes(),
	}

	// store the captured request in the storage
	reqMeta, rmErr := h.s.NewRequest(r.Context(), sID, rID, captured)
	if rmErr != nil {
		renderError(h.log, w, r, rmErr)

		return
	}

	w.Header().Set("X-Wh-Request-Id", rID)
	w.Header().Set("Content-Length", strconv.Itoa(len(s.Response.Body)))

	// set the header to allow CORS requests from any origin and method
	w.Header().Set("Access-Control-Allow-Origin", "*")
	w.Header().Set("Access-Control-Allow-Methods", "*")
	w.Header().Set("Access-Control-Allow-Headers", "*")

	// publish the event about the captured request in the background.
	// wrapping context via WithoutCancel is important here - the request context will be canceled as soon as the
	// handler returns, and we don't want that - we need to publish the event even if the request has already been
	// completed.
	go func(pCtx context.Context) {
		hdrs := make([]pubsub.HttpHeader, len(captured.Headers))
		for i, hrd := range captured.Headers {
			hdrs[i] = pubsub.HttpHeader(hrd)
		}

		if err := h.ps.Publish(pCtx, sID, pubsub.RequestEvent{
			Action: pubsub.RequestActionCreate,
			Request: &pubsub.RequestData{
				ID:         rID,
				ClientAddr: captured.ClientAddr,
				Method:     captured.Method,
				Headers:    hdrs,
				URL:        captured.URL,
				CreatedAt:  reqMeta.CreatedAt,
			},
		}); err != nil {
			h.log.Error("Failed to publish a captured request", logger.Error(err))
		}
	}(context.WithoutCancel(r.Context()))

	// emulate the response delay, if specified in the session settings
	if s.Response.Delay > 0 {
		var timer = time.NewTimer(s.Response.Delay)
		defer timer.Stop()

		select {
		case <-r.Context().Done():
		case <-timer.C:
		}
	}

	// set HTTP response headers from the session settings
	for _, hdr := range s.Response.Headers {
		w.Header().Set(hdr.Name, hdr.Value)
	}

	status := s.Response.Code
	if reqStatus, ok := extractRequestedStatusCode(r); ok {
		status = reqStatus
	}

	w.WriteHeader(int(status))

	if _, err := w.Write(s.Response.Body); err != nil { //nolint:gosec
		h.log.Error("Failed to write the response body", logger.Error(err))
	}
}

// extractFullUrl returns the full URL from the request.
func extractFullUrl(r *http.Request) string {
	var scheme = "http"
	if r.TLS != nil {
		scheme = "https"
	}

	return fmt.Sprintf("%s://%s%s", scheme, r.Host, r.RequestURI)
}

// extractRequestedStatusCode returns the requested HTTP status code from the URL path, if it is present and valid.
func extractRequestedStatusCode(r *http.Request) (uint16, bool) {
	if r == nil || r.URL == nil {
		return 0, false
	}

	code, err := strconv.Atoi(path.Base(r.URL.Path))
	if err != nil || code < 100 || code > 599 {
		return 0, false
	}

	return uint16(code), true
}

// extractSessionID extracts the session ID from the request.
//
// It returns the session ID and a boolean indicating whether the extraction was successful.
func extractSessionID(r *http.Request) (string, bool) {
	if r == nil || r.URL == nil {
		return "", false
	}

	clean := strings.TrimLeft(r.URL.Path, "/")

	const uuidLen = 36

	if len(clean) >= uuidLen {
		// fast check for UUID format: 8-4-4-4-12
		if clean[8] != '-' || clean[13] != '-' || clean[18] != '-' || clean[23] != '-' {
			return "", false
		}

		// check all characters to be valid UUID chars (hex digits and dashes)
		for _, c := range clean[:uuidLen] { //nolint:staticcheck
			if !(c == '-' || (c >= '0' && c <= '9') || (c >= 'a' && c <= 'f') || (c >= 'A' && c <= 'F')) {
				return "", false
			}
		}

		return clean[:uuidLen], true
	}

	return "", false
}
