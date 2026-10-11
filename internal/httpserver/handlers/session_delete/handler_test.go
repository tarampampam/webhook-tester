package session_delete_test

import (
	"context"
	"errors"
	"testing"
	"time"

	"gh.tarampamp.am/webhook-tester/v3/internal/httpserver/handlers/session_delete"
	"gh.tarampamp.am/webhook-tester/v3/internal/httpserver/openapi"
	"gh.tarampamp.am/webhook-tester/v3/internal/storage"
	"gh.tarampamp.am/webhook-tester/v3/internal/testutil/assert"
)

func TestHandler_Handle(t *testing.T) {
	t.Parallel()

	const fixedSID = "550e8400-e29b-41d4-a716-446655440000"

	for name, tt := range map[string]struct {
		giveID   string
		mockFn   func(context.Context, string) error
		wantErr  bool
		checkErr func(*testing.T, error)
		check    func(*testing.T, *openapi.SuccessfulOperationResponse)
	}{
		"success/session deleted": {
			giveID: fixedSID,
			mockFn: func(_ context.Context, id string) error {
				assert.Equal(t, fixedSID, id)

				return nil
			},
			check: func(t *testing.T, resp *openapi.SuccessfulOperationResponse) {
				assert.True(t, resp.Success)
			},
		},

		"error/session not found returns openapi not found": {
			giveID:  fixedSID,
			mockFn:  func(_ context.Context, _ string) error { return storage.ErrSessionNotFound },
			wantErr: true,
			checkErr: func(t *testing.T, err error) {
				assert.ErrorIs(t, err, openapi.ErrNotFound)
			},
		},

		"error/storage failure wrapped": {
			giveID:  fixedSID,
			mockFn:  func(_ context.Context, _ string) error { return errors.New("storage unavailable") },
			wantErr: true,
			checkErr: func(t *testing.T, err error) {
				assert.ErrorContains(t, err, "delete session")
				assert.ErrorContains(t, err, "storage unavailable")
			},
		},
	} {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			h := session_delete.New(&mockStorage{deleteSessionFn: tt.mockFn})
			resp, err := h.Handle(t.Context(), tt.giveID)

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
	deleteSessionFn func(context.Context, string) error
}

var _ storage.SessionStorage = (*mockStorage)(nil)

func (m *mockStorage) NewSession(context.Context, string, storage.SessionResponse, time.Duration) (*storage.SessionMeta, error) {
	panic("mockStorage: NewSession not expected in this handler")
}

func (m *mockStorage) GetSession(context.Context, string) (*storage.Session, error) {
	panic("mockStorage: GetSession not expected in this handler")
}

func (m *mockStorage) AddSessionTTL(context.Context, string, time.Duration) error {
	panic("mockStorage: AddSessionTTL not expected in this handler")
}

func (m *mockStorage) DeleteSession(ctx context.Context, sID string) error {
	if m.deleteSessionFn == nil {
		panic("mockStorage: unexpected DeleteSession call")
	}

	return m.deleteSessionFn(ctx, sID)
}
