package session_check_exists

import (
	"context"
	"errors"
	"sync"

	"golang.org/x/sync/errgroup"

	"gh.tarampamp.am/webhook-tester/v2/internal/http/openapi"
	"gh.tarampamp.am/webhook-tester/v2/internal/storage"
)

type Handler struct{ db storage.Storage }

func New(db storage.Storage) *Handler { return &Handler{db: db} }

func (h *Handler) Handle(ctx context.Context, ids []openapi.UUID) (*openapi.CheckSessionExistsResponse, error) {
	var (
		eg, egCtx = errgroup.WithContext(ctx)

		mu  sync.Mutex
		res = make(openapi.CheckSessionExistsResponse, len(ids)) // map[sID]bool
	)

	for _, id := range ids {
		eg.Go(func(sID string) func() error {
			return func() error {
				if _, err := h.db.GetSession(egCtx, sID); err != nil {
					if errors.Is(err, storage.ErrNotFound) {
						mu.Lock()
						res[sID] = false
						mu.Unlock()

						return nil
					}

					return err
				}

				mu.Lock()
				res[sID] = true
				mu.Unlock()

				return nil
			}
		}(id.String()))
	}

	if err := eg.Wait(); err != nil {
		return nil, err
	}

	return &res, nil
}
