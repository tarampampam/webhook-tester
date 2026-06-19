package version_latest

import (
	"context"
	"fmt"
	"sync"
	"time"

	"gh.tarampamp.am/webhook-tester/v3/internal/httpserver/openapi"
)

type latestVersionProvider func(context.Context) (string, error)

// Handler fetches and caches the latest application version from an upstream provider.
type Handler struct {
	p latestVersionProvider

	mu        sync.Mutex // protects the fields below
	updatedAt time.Time
	cache     string
}

// New creates a new Handler using p as the upstream version provider.
func New(p latestVersionProvider) *Handler { return &Handler{p: p} }

// Handle returns the latest version. The result is cached for 5 minutes - after expiry
// the provider is called again to refresh. Safe for concurrent use.
func (h *Handler) Handle(ctx context.Context) (*openapi.VersionResponse, error) {
	const cacheTTL = 5 * time.Minute

	h.mu.Lock()
	defer h.mu.Unlock()

	if time.Since(h.updatedAt) < cacheTTL && h.cache != "" {
		return &openapi.VersionResponse{Version: h.cache}, nil
	}

	version, fetchErr := h.p(ctx)
	if fetchErr != nil {
		return nil, fmt.Errorf("failed to fetch the latest version: %w", fetchErr)
	}

	h.updatedAt, h.cache = time.Now(), version

	return &openapi.VersionResponse{Version: version}, nil
}
