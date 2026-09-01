package session_check_exists_test

import (
	"context"
	"errors"
	"testing"
	"time"

	"gh.tarampamp.am/webhook-tester/v3/internal/httpserver/handlers/session_check_exists"
	"gh.tarampamp.am/webhook-tester/v3/internal/httpserver/openapi"
	"gh.tarampamp.am/webhook-tester/v3/internal/storage"
	"gh.tarampamp.am/webhook-tester/v3/internal/testutil/assert"
)

func TestHandler_Handle(t *testing.T) {
	t.Parallel()

	const (
		id1 = "550e8400-e29b-41d4-a716-446655440001"
		id2 = "550e8400-e29b-41d4-a716-446655440002"
		id3 = "550e8400-e29b-41d4-a716-446655440003"
	)

	existingSession := &storage.Session{
		Response: storage.SessionResponse{Code: 200},
		Meta:     storage.SessionMeta{CreatedAt: time.Now()},
	}

	for name, tt := range map[string]struct {
		giveIDs  []string
		mockFn   func(context.Context, string) (*storage.Session, error)
		wantErr  bool
		checkErr func(*testing.T, error)
		check    func(*testing.T, *openapi.CheckSessionExistsResponse)
	}{
		"success/all ids exist": {
			giveIDs: []string{id1, id2},
			mockFn: func(_ context.Context, _ string) (*storage.Session, error) {
				return existingSession, nil
			},
			check: func(t *testing.T, resp *openapi.CheckSessionExistsResponse) {
				assert.DeepEqual(t, openapi.CheckSessionExistsResponse{id1: true, id2: true}, *resp)
			},
		},

		"success/none exist": {
			giveIDs: []string{id1, id2},
			mockFn: func(_ context.Context, _ string) (*storage.Session, error) {
				return nil, storage.ErrSessionNotFound
			},
			check: func(t *testing.T, resp *openapi.CheckSessionExistsResponse) {
				assert.DeepEqual(t, openapi.CheckSessionExistsResponse{id1: false, id2: false}, *resp)
			},
		},

		"success/mixed existence": {
			giveIDs: []string{id1, id2, id3},
			mockFn: func(_ context.Context, id string) (*storage.Session, error) {
				if id == id2 {
					return nil, storage.ErrSessionNotFound
				}

				return existingSession, nil
			},
			check: func(t *testing.T, resp *openapi.CheckSessionExistsResponse) {
				assert.DeepEqual(t, openapi.CheckSessionExistsResponse{id1: true, id2: false, id3: true}, *resp)
			},
		},

		"error/storage failure returns error": {
			giveIDs: []string{id1},
			mockFn: func(_ context.Context, _ string) (*storage.Session, error) {
				return nil, errors.New("storage unavailable")
			},
			wantErr: true,
			checkErr: func(t *testing.T, err error) {
				assert.ErrorContains(t, err, "storage unavailable")
			},
		},

		"error/unexpected error among multiple ids": {
			giveIDs: []string{id1, id2},
			mockFn: func(_ context.Context, id string) (*storage.Session, error) {
				if id == id1 {
					return nil, errors.New("disk failure")
				}

				return existingSession, nil
			},
			wantErr: true,
			checkErr: func(t *testing.T, err error) {
				assert.ErrorContains(t, err, "disk failure")
			},
		},
	} {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			h := session_check_exists.New(&mockStorage{getSessionFn: tt.mockFn})
			resp, err := h.Handle(t.Context(), tt.giveIDs)

			if tt.wantErr {
				assert.Error(t, err)
				assert.Nil(t, resp)

				if tt.checkErr != nil {
					tt.checkErr(t, err)
				}
			} else {
				assert.NoError(t, err)
				assert.NotNil(t, resp)

				if tt.check != nil {
					tt.check(t, resp)
				}
			}
		})
	}
}

// --------------------------------------------------------------------------------------------------------------------

// mockStorage implements [storage.SessionStorage] for handler isolation tests.
type mockStorage struct {
	getSessionFn func(context.Context, string) (*storage.Session, error)
}

var _ storage.SessionStorage = (*mockStorage)(nil)

func (m *mockStorage) NewSession(context.Context, string, storage.SessionResponse, time.Duration) (*storage.SessionMeta, error) {
	panic("mockStorage: NewSession not expected in this handler")
}

func (m *mockStorage) GetSession(ctx context.Context, sID string) (*storage.Session, error) {
	if m.getSessionFn == nil {
		panic("mockStorage: unexpected GetSession call")
	}

	return m.getSessionFn(ctx, sID)
}

func (m *mockStorage) AddSessionTTL(context.Context, string, time.Duration) error {
	panic("mockStorage: AddSessionTTL not expected in this handler")
}

func (m *mockStorage) DeleteSession(context.Context, string) error {
	panic("mockStorage: DeleteSession not expected in this handler")
}
