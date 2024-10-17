package version_latest

import (
	"fmt"
	"net/http"
	"sync"
	"time"

	"gh.tarampamp.am/webhook-tester/v2/internal/http/openapi"
)

type (
	versionFetcher func() (string, error)

	Handler struct {
		mu        sync.Mutex // protects the fields below
		updatedAt time.Time
		cache     string

		fetcher versionFetcher
	}
)

func New(fetcher versionFetcher) *Handler { return &Handler{fetcher: fetcher} }

func (h *Handler) Handle(w http.ResponseWriter) (*openapi.VersionResponse, error) {
	const cacheTTL, cacheHitHeader = 5 * time.Minute, "X-Cache"

	h.mu.Lock()
	defer h.mu.Unlock()

	// check if the cache is still valid
	if time.Since(h.updatedAt) < cacheTTL && h.cache != "" {
		w.Header().Set(cacheHitHeader, "HIT")

		// return the cached value
		return &openapi.VersionResponse{Version: h.cache}, nil
	}

	w.Header().Set(cacheHitHeader, "MISS")

	// fetch the latest version
	version, fetchErr := h.fetcher()
	if fetchErr != nil {
		return nil, fmt.Errorf("failed to fetch the latest version: %w", fetchErr)
	}

	// update the cache and the timestamp
	h.updatedAt, h.cache = time.Now(), version

	return &openapi.VersionResponse{Version: version}, nil
}
