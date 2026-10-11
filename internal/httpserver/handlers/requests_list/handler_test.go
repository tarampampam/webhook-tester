package requests_list_test

import (
	"context"
	"encoding/base64"
	"errors"
	"iter"
	"testing"
	"time"

	"gh.tarampamp.am/webhook-tester/v3/internal/httpserver/handlers/requests_list"
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
		mockFn   func(context.Context, string, *error) (iter.Seq2[string, storage.Request], error)
		wantErr  bool
		checkErr func(*testing.T, error)
		check    func(*testing.T, *openapi.CapturedRequestsListResponse)
	}{
		"success/empty session returns empty slice": {
			giveSID: fixedSID,
			mockFn: func(_ context.Context, _ string, _ *error) (iter.Seq2[string, storage.Request], error) {
				return func(yield func(string, storage.Request) bool) {}, nil
			},
			check: func(t *testing.T, resp *openapi.CapturedRequestsListResponse) {
				assert.Equal(t, 0, len(*resp))
			},
		},

		"success/single request with all fields mapped": {
			giveSID: fixedSID,
			mockFn: func(_ context.Context, sID string, _ *error) (iter.Seq2[string, storage.Request], error) {
				assert.Equal(t, fixedSID, sID)

				return func(yield func(string, storage.Request) bool) {
					yield(fixedRID, storage.Request{
						Data: storage.CapturedRequest{
							ClientAddr: "1.2.3.4",
							Method:     "POST",
							URL:        "https://example.com/hook",
						},
						Meta: storage.RequestMeta{CreatedAt: fixedNow},
					})
				}, nil
			},
			check: func(t *testing.T, resp *openapi.CapturedRequestsListResponse) {
				assert.Equal(t, 1, len(*resp))

				got := (*resp)[0]
				assert.Equal(t, fixedRID, got.Uuid)
				assert.Equal(t, fixedNow.UnixMilli(), got.CapturedAtUnixMilli)
				assert.Equal(t, "POST", got.Method)
				assert.Equal(t, "https://example.com/hook", got.Url)
				assert.Equal(t, "1.2.3.4", got.ClientAddress)
				assert.DeepEqual(t, []openapi.HttpHeader{}, got.Headers)
				assert.Equal(t, "", got.RequestPayloadBase64)
			},
		},

		"success/multiple requests preserve iteration order": {
			giveSID: fixedSID,
			mockFn: func(_ context.Context, _ string, _ *error) (iter.Seq2[string, storage.Request], error) {
				items := []struct {
					id  string
					req storage.Request
				}{
					{"id-1", storage.Request{Data: storage.CapturedRequest{Method: "GET"}, Meta: storage.RequestMeta{CreatedAt: fixedNow}}},
					{"id-2", storage.Request{Data: storage.CapturedRequest{Method: "POST"}, Meta: storage.RequestMeta{CreatedAt: fixedNow}}},
					{"id-3", storage.Request{Data: storage.CapturedRequest{Method: "DELETE"}, Meta: storage.RequestMeta{CreatedAt: fixedNow}}},
				}

				return func(yield func(string, storage.Request) bool) {
					for _, item := range items {
						if !yield(item.id, item.req) {
							return
						}
					}
				}, nil
			},
			check: func(t *testing.T, resp *openapi.CapturedRequestsListResponse) {
				assert.Equal(t, 3, len(*resp))
				assert.Equal(t, "id-1", (*resp)[0].Uuid)
				assert.Equal(t, "id-2", (*resp)[1].Uuid)
				assert.Equal(t, "id-3", (*resp)[2].Uuid)
			},
		},

		"success/headers converted from storage to openapi format": {
			giveSID: fixedSID,
			mockFn: func(_ context.Context, _ string, _ *error) (iter.Seq2[string, storage.Request], error) {
				return func(yield func(string, storage.Request) bool) {
					yield(fixedRID, storage.Request{
						Data: storage.CapturedRequest{
							Method: "GET",
							Headers: []storage.RequestHeader{
								{Name: "Content-Type", Value: "application/json"},
								{Name: "X-Custom", Value: "value"},
							},
						},
						Meta: storage.RequestMeta{CreatedAt: fixedNow},
					})
				}, nil
			},
			check: func(t *testing.T, resp *openapi.CapturedRequestsListResponse) {
				assert.DeepEqual(t, []openapi.HttpHeader{
					{Name: "Content-Type", Value: "application/json"},
					{Name: "X-Custom", Value: "value"},
				}, (*resp)[0].Headers)
			},
		},

		"success/body bytes encoded to base64": {
			giveSID: fixedSID,
			mockFn: func(_ context.Context, _ string, _ *error) (iter.Seq2[string, storage.Request], error) {
				return func(yield func(string, storage.Request) bool) {
					yield(fixedRID, storage.Request{
						Data: storage.CapturedRequest{Method: "POST", Body: []byte("hello body")},
						Meta: storage.RequestMeta{CreatedAt: fixedNow},
					})
				}, nil
			},
			check: func(t *testing.T, resp *openapi.CapturedRequestsListResponse) {
				assert.Equal(t, base64.StdEncoding.EncodeToString([]byte("hello body")), (*resp)[0].RequestPayloadBase64)
			},
		},

		"error/session not found returns openapi not found": {
			giveSID: fixedSID,
			mockFn: func(_ context.Context, _ string, _ *error) (iter.Seq2[string, storage.Request], error) {
				return nil, storage.ErrSessionNotFound
			},
			wantErr: true,
			checkErr: func(t *testing.T, err error) {
				assert.ErrorIs(t, err, openapi.ErrNotFound)
			},
		},

		"error/setup failure wrapped": {
			giveSID: fixedSID,
			mockFn: func(_ context.Context, _ string, _ *error) (iter.Seq2[string, storage.Request], error) {
				return nil, errors.New("storage unavailable")
			},
			wantErr: true,
			checkErr: func(t *testing.T, err error) {
				assert.ErrorContains(t, err, "get requests")
				assert.ErrorContains(t, err, "storage unavailable")
			},
		},

		"error/mid-iteration failure wrapped": {
			giveSID: fixedSID,
			mockFn: func(_ context.Context, _ string, iterErr *error) (iter.Seq2[string, storage.Request], error) {
				return func(yield func(string, storage.Request) bool) {
					*iterErr = errors.New("disk failure")
				}, nil
			},
			wantErr: true,
			checkErr: func(t *testing.T, err error) {
				assert.ErrorContains(t, err, "iterate requests")
				assert.ErrorContains(t, err, "disk failure")
			},
		},
	} {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			h := requests_list.New(&mockStorage{getRequestsFn: tt.mockFn})
			resp, err := h.Handle(t.Context(), tt.giveSID)

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
	getRequestsFn func(context.Context, string, *error) (iter.Seq2[string, storage.Request], error)
}

var _ storage.RequestStorage = (*mockStorage)(nil)

func (m *mockStorage) NewRequest(context.Context, string, string, storage.CapturedRequest) (*storage.RequestMeta, error) {
	panic("mockStorage: NewRequest not expected in this handler")
}

func (m *mockStorage) GetRequest(context.Context, string, string) (*storage.Request, error) {
	panic("mockStorage: GetRequest not expected in this handler")
}

func (m *mockStorage) GetRequests(ctx context.Context, sID string, iterErr *error) (iter.Seq2[string, storage.Request], error) {
	if m.getRequestsFn == nil {
		panic("mockStorage: unexpected GetRequests call")
	}

	return m.getRequestsFn(ctx, sID, iterErr)
}

func (m *mockStorage) DeleteRequest(context.Context, string, string) error {
	panic("mockStorage: DeleteRequest not expected in this handler")
}

func (m *mockStorage) DeleteAllRequests(context.Context, string) error {
	panic("mockStorage: DeleteAllRequests not expected in this handler")
}
