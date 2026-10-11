package session_check_exists

import (
	"context"
	"errors"
	"sync"

	"gh.tarampamp.am/webhook-tester/v3/internal/errgroup"
	"gh.tarampamp.am/webhook-tester/v3/internal/httpserver/openapi"
	"gh.tarampamp.am/webhook-tester/v3/internal/storage"
)

// Handler handles HTTP requests for checking whether sessions exist.
type Handler struct {
	s storage.SessionStorage
}

// New creates a new Handler.
func New(s storage.SessionStorage) *Handler { return &Handler{s: s} }

// Handle checks which of the given session IDs exist and returns a map of ID to existence flag.
func (h *Handler) Handle(ctx context.Context, ids []string) (*openapi.CheckSessionExistsResponse, error) {
	var (
		eg, _ = errgroup.New(ctx)

		mu  sync.Mutex
		res = make(openapi.CheckSessionExistsResponse, len(ids))
	)

	for _, id := range ids {
		eg.Go(func(ctx context.Context) error {
			_, err := h.s.GetSession(ctx, id) // isn't optimal, but I'm too lazy to add a separate "checking existence method"
			if err != nil {
				if errors.Is(err, storage.ErrSessionNotFound) {
					mu.Lock()
					res[id] = false
					mu.Unlock()

					return nil
				}

				return err
			}

			mu.Lock()
			res[id] = true
			mu.Unlock()

			return nil
		})
	}

	if err := eg.Wait(); err != nil {
		return nil, err
	}

	return &res, nil
}
