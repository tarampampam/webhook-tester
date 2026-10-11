package request_delete_test

import (
	"context"
	"errors"
	"iter"
	"testing"

	"gh.tarampamp.am/webhook-tester/v3/internal/httpserver/handlers/request_delete"
	"gh.tarampamp.am/webhook-tester/v3/internal/httpserver/openapi"
	"gh.tarampamp.am/webhook-tester/v3/internal/pubsub"
	"gh.tarampamp.am/webhook-tester/v3/internal/storage"
	"gh.tarampamp.am/webhook-tester/v3/internal/testutil/assert"
)

func TestHandler_Handle(t *testing.T) {
	t.Parallel()

	const (
		fixedSID = "550e8400-e29b-41d4-a716-446655440000"
		fixedRID = "660e8400-e29b-41d4-a716-446655440001"
	)

	for name, tt := range map[string]struct {
		giveSID   string
		giveRID   string
		deleteFn  func(context.Context, string, string) error
		publishFn func(context.Context, string, pubsub.RequestEvent) error
		wantErr   bool
		checkErr  func(*testing.T, error)
		check     func(*testing.T, *openapi.SuccessfulOperationResponse)
	}{
		"success/request deleted and event published": {
			giveSID: fixedSID,
			giveRID: fixedRID,
			deleteFn: func(_ context.Context, sID, rID string) error {
				assert.Equal(t, fixedSID, sID)
				assert.Equal(t, fixedRID, rID)

				return nil
			},
			publishFn: func(_ context.Context, topic string, event pubsub.RequestEvent) error {
				assert.Equal(t, fixedSID, topic)
				assert.Equal(t, pubsub.RequestActionDelete, event.Action)
				assert.Equal(t, fixedRID, event.Request.ID)

				return nil
			},
			check: func(t *testing.T, resp *openapi.SuccessfulOperationResponse) {
				assert.True(t, resp.Success)
			},
		},

		"error/session not found returns openapi not found": {
			giveSID:  fixedSID,
			giveRID:  fixedRID,
			deleteFn: func(_ context.Context, _, _ string) error { return storage.ErrSessionNotFound },
			wantErr:  true,
			checkErr: func(t *testing.T, err error) {
				assert.ErrorIs(t, err, openapi.ErrNotFound)
			},
		},

		"error/request not found returns openapi not found": {
			giveSID:  fixedSID,
			giveRID:  fixedRID,
			deleteFn: func(_ context.Context, _, _ string) error { return storage.ErrRequestNotFound },
			wantErr:  true,
			checkErr: func(t *testing.T, err error) {
				assert.ErrorIs(t, err, openapi.ErrNotFound)
			},
		},

		"error/storage failure wrapped": {
			giveSID:  fixedSID,
			giveRID:  fixedRID,
			deleteFn: func(_ context.Context, _, _ string) error { return errors.New("storage unavailable") },
			wantErr:  true,
			checkErr: func(t *testing.T, err error) {
				assert.ErrorContains(t, err, "delete request")
				assert.ErrorContains(t, err, "storage unavailable")
			},
		},

		"error/publish failure wrapped": {
			giveSID:   fixedSID,
			giveRID:   fixedRID,
			deleteFn:  func(_ context.Context, _, _ string) error { return nil },
			publishFn: func(_ context.Context, _ string, _ pubsub.RequestEvent) error { return errors.New("bus down") },
			wantErr:   true,
			checkErr: func(t *testing.T, err error) {
				assert.ErrorContains(t, err, "publish event")
				assert.ErrorContains(t, err, "bus down")
			},
		},
	} {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			h := request_delete.New(
				&mockStorage{deleteRequestFn: tt.deleteFn},
				&mockPublisher{publishFn: tt.publishFn},
			)
			resp, err := h.Handle(t.Context(), tt.giveSID, tt.giveRID)

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

type mockStorage struct {
	deleteRequestFn func(context.Context, string, string) error
}

var _ storage.RequestStorage = (*mockStorage)(nil)

func (m *mockStorage) NewRequest(context.Context, string, string, storage.CapturedRequest) (*storage.RequestMeta, error) {
	panic("mockStorage: NewRequest not expected in this handler")
}

func (m *mockStorage) GetRequest(context.Context, string, string) (*storage.Request, error) {
	panic("mockStorage: GetRequest not expected in this handler")
}

func (m *mockStorage) GetRequests(context.Context, string, *error) (iter.Seq2[string, storage.Request], error) {
	panic("mockStorage: GetRequests not expected in this handler")
}

func (m *mockStorage) DeleteRequest(ctx context.Context, sID, rID string) error {
	if m.deleteRequestFn == nil {
		panic("mockStorage: unexpected DeleteRequest call")
	}

	return m.deleteRequestFn(ctx, sID, rID)
}

func (m *mockStorage) DeleteAllRequests(context.Context, string) error {
	panic("mockStorage: DeleteAllRequests not expected in this handler")
}

// --------------------------------------------------------------------------------------------------------------------

type mockPublisher struct {
	publishFn func(context.Context, string, pubsub.RequestEvent) error
}

var _ pubsub.Publisher = (*mockPublisher)(nil)

func (m *mockPublisher) Publish(ctx context.Context, topic string, event pubsub.RequestEvent) error {
	if m.publishFn == nil {
		panic("mockPublisher: unexpected Publish call")
	}

	return m.publishFn(ctx, topic, event)
}
