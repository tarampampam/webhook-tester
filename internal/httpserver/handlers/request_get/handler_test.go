package request_get_test

import (
	"context"
	"encoding/base64"
	"errors"
	"iter"
	"testing"
	"time"

	"gh.tarampamp.am/webhook-tester/v3/internal/httpserver/handlers/request_get"
	"gh.tarampamp.am/webhook-tester/v3/internal/httpserver/openapi"
	"gh.tarampamp.am/webhook-tester/v3/internal/storage"
	"gh.tarampamp.am/webhook-tester/v3/internal/testutil/assert"
)

func TestHandler_Handle(t *testing.T) {
	t.Parallel()

	var (
		fixedNow = time.Date(2024, 6, 15, 12, 0, 0, 0, time.UTC)
		fixedSID = "550e8400-e29b-41d4-a716-446655440000"
		fixedRID = "660e8400-e29b-41d4-a716-446655440001"
	)

	for name, tt := range map[string]struct {
		giveSID  string
		giveRID  string
		mockFn   func(context.Context, string, string) (*storage.Request, error)
		wantErr  bool
		checkErr func(*testing.T, error)
		check    func(*testing.T, *openapi.CapturedRequestsResponse)
	}{
		"success/minimal request": {
			giveSID: fixedSID,
			giveRID: fixedRID,
			mockFn: func(_ context.Context, sID, rID string) (*storage.Request, error) {
				assert.Equal(t, fixedSID, sID)
				assert.Equal(t, fixedRID, rID)

				return &storage.Request{
					Data: storage.CapturedRequest{Method: "GET", URL: "https://example.com/hook"},
					Meta: storage.RequestMeta{CreatedAt: fixedNow},
				}, nil
			},
			check: func(t *testing.T, resp *openapi.CapturedRequestsResponse) {
				assert.Equal(t, fixedRID, resp.Uuid)
				assert.Equal(t, fixedNow.UnixMilli(), resp.CapturedAtUnixMilli)
				assert.Equal(t, "GET", resp.Method)
				assert.Equal(t, "https://example.com/hook", resp.Url)
				assert.Equal(t, "", resp.ClientAddress)
				assert.DeepEqual(t, []openapi.HttpHeader{}, resp.Headers)
				assert.Equal(t, "", resp.RequestPayloadBase64)
			},
		},

		"success/body bytes encoded to base64": {
			giveSID: fixedSID,
			giveRID: fixedRID,
			mockFn: func(_ context.Context, _, _ string) (*storage.Request, error) {
				return &storage.Request{
					Data: storage.CapturedRequest{Method: "POST", Body: []byte("hello body")},
					Meta: storage.RequestMeta{CreatedAt: fixedNow},
				}, nil
			},
			check: func(t *testing.T, resp *openapi.CapturedRequestsResponse) {
				assert.Equal(t, base64.StdEncoding.EncodeToString([]byte("hello body")), resp.RequestPayloadBase64)
			},
		},

		"success/headers converted from storage to openapi format": {
			giveSID: fixedSID,
			giveRID: fixedRID,
			mockFn: func(_ context.Context, _, _ string) (*storage.Request, error) {
				return &storage.Request{
					Data: storage.CapturedRequest{
						Method: "GET",
						Headers: []storage.RequestHeader{
							{Name: "Content-Type", Value: "application/json"},
							{Name: "X-Custom", Value: "value"},
						},
					},
					Meta: storage.RequestMeta{CreatedAt: fixedNow},
				}, nil
			},
			check: func(t *testing.T, resp *openapi.CapturedRequestsResponse) {
				assert.DeepEqual(t, []openapi.HttpHeader{
					{Name: "Content-Type", Value: "application/json"},
					{Name: "X-Custom", Value: "value"},
				}, resp.Headers)
			},
		},

		"error/session not found returns openapi not found": {
			giveSID: fixedSID,
			giveRID: fixedRID,
			mockFn: func(_ context.Context, _, _ string) (*storage.Request, error) {
				return nil, storage.ErrSessionNotFound
			},
			wantErr: true,
			checkErr: func(t *testing.T, err error) {
				assert.ErrorIs(t, err, openapi.ErrNotFound)
			},
		},

		"error/request not found returns openapi not found": {
			giveSID: fixedSID,
			giveRID: fixedRID,
			mockFn: func(_ context.Context, _, _ string) (*storage.Request, error) {
				return nil, storage.ErrRequestNotFound
			},
			wantErr: true,
			checkErr: func(t *testing.T, err error) {
				assert.ErrorIs(t, err, openapi.ErrNotFound)
			},
		},

		"error/storage failure wrapped": {
			giveSID: fixedSID,
			giveRID: fixedRID,
			mockFn: func(_ context.Context, _, _ string) (*storage.Request, error) {
				return nil, errors.New("storage unavailable")
			},
			wantErr: true,
			checkErr: func(t *testing.T, err error) {
				assert.ErrorContains(t, err, "get request")
				assert.ErrorContains(t, err, "storage unavailable")
			},
		},
	} {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			h := request_get.New(&mockStorage{getRequestFn: tt.mockFn})
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

// mockStorage implements [storage.RequestStorage] for handler isolation tests.
type mockStorage struct {
	getRequestFn func(context.Context, string, string) (*storage.Request, error)
}

var _ storage.RequestStorage = (*mockStorage)(nil)

func (m *mockStorage) NewRequest(context.Context, string, string, storage.CapturedRequest) (*storage.RequestMeta, error) {
	panic("mockStorage: NewRequest not expected in this handler")
}

func (m *mockStorage) GetRequest(ctx context.Context, sID, rID string) (*storage.Request, error) {
	if m.getRequestFn == nil {
		panic("mockStorage: unexpected GetRequest call")
	}

	return m.getRequestFn(ctx, sID, rID)
}

func (m *mockStorage) GetRequests(context.Context, string, *error) (iter.Seq2[string, storage.Request], error) {
	panic("mockStorage: GetRequests not expected in this handler")
}

func (m *mockStorage) DeleteRequest(context.Context, string, string) error {
	panic("mockStorage: DeleteRequest not expected in this handler")
}

func (m *mockStorage) DeleteAllRequests(context.Context, string) error {
	panic("mockStorage: DeleteAllRequests not expected in this handler")
}
