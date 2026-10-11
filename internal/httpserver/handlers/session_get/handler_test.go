package session_get_test

import (
	"context"
	"encoding/base64"
	"errors"
	"testing"
	"time"

	"gh.tarampamp.am/webhook-tester/v3/internal/httpserver/handlers/session_get"
	"gh.tarampamp.am/webhook-tester/v3/internal/httpserver/openapi"
	"gh.tarampamp.am/webhook-tester/v3/internal/storage"
	"gh.tarampamp.am/webhook-tester/v3/internal/testutil/assert"
)

func TestHandler_Handle(t *testing.T) {
	t.Parallel()

	var (
		fixedNow = time.Date(2024, 6, 15, 12, 0, 0, 0, time.UTC)
		fixedSID = "550e8400-e29b-41d4-a716-446655440000"
	)

	for name, tt := range map[string]struct {
		giveReq  string
		mockFn   func(context.Context, string) (*storage.Session, error)
		wantErr  bool
		checkErr func(*testing.T, error)
		check    func(*testing.T, *openapi.SessionOptionsResponse)
	}{
		"success/minimal session": {
			giveReq: fixedSID,
			mockFn: func(_ context.Context, _ string) (*storage.Session, error) {
				return &storage.Session{
					Response: storage.SessionResponse{Code: 200},
					Meta:     storage.SessionMeta{CreatedAt: fixedNow},
				}, nil
			},
			check: func(t *testing.T, resp *openapi.SessionOptionsResponse) {
				assert.Equal(t, fixedSID, resp.Uuid)
				assert.Equal(t, fixedNow.UnixMilli(), resp.CreatedAtUnixMilli)
				assert.Equal(t, 200, resp.Response.StatusCode)
				assert.Equal(t, uint16(0), resp.Response.Delay)
				assert.DeepEqual(t, []openapi.HttpHeader{}, resp.Response.Headers)
				assert.Equal(t, "", resp.Response.ResponseBodyBase64)
			},
		},

		"success/sID passed to storage": {
			giveReq: fixedSID,
			mockFn: func(_ context.Context, id string) (*storage.Session, error) {
				assert.Equal(t, fixedSID, id)

				return &storage.Session{
					Response: storage.SessionResponse{Code: 200},
					Meta:     storage.SessionMeta{CreatedAt: fixedNow},
				}, nil
			},
		},

		"success/body bytes encoded to base64": {
			giveReq: fixedSID,
			mockFn: func(_ context.Context, _ string) (*storage.Session, error) {
				return &storage.Session{
					Response: storage.SessionResponse{Code: 200, Body: []byte("hello body")},
					Meta:     storage.SessionMeta{CreatedAt: fixedNow},
				}, nil
			},
			check: func(t *testing.T, resp *openapi.SessionOptionsResponse) {
				assert.Equal(t, base64.StdEncoding.EncodeToString([]byte("hello body")), resp.Response.ResponseBodyBase64)
			},
		},

		"success/nil body encoded as empty string": {
			giveReq: fixedSID,
			mockFn: func(_ context.Context, _ string) (*storage.Session, error) {
				return &storage.Session{
					Response: storage.SessionResponse{Code: 200, Body: nil},
					Meta:     storage.SessionMeta{CreatedAt: fixedNow},
				}, nil
			},
			check: func(t *testing.T, resp *openapi.SessionOptionsResponse) {
				assert.Equal(t, "", resp.Response.ResponseBodyBase64)
			},
		},

		"success/headers converted from storage to openapi format": {
			giveReq: fixedSID,
			mockFn: func(_ context.Context, _ string) (*storage.Session, error) {
				return &storage.Session{
					Response: storage.SessionResponse{
						Code: 200,
						Headers: []storage.ResponseHeader{
							{Name: "Content-Type", Value: "application/json"},
							{Name: "X-Custom", Value: "value"},
						},
					},
					Meta: storage.SessionMeta{CreatedAt: fixedNow},
				}, nil
			},
			check: func(t *testing.T, resp *openapi.SessionOptionsResponse) {
				assert.DeepEqual(t, []openapi.HttpHeader{
					{Name: "Content-Type", Value: "application/json"},
					{Name: "X-Custom", Value: "value"},
				}, resp.Response.Headers)
			},
		},

		"success/delay converted from duration to uint16 seconds": {
			giveReq: fixedSID,
			mockFn: func(_ context.Context, _ string) (*storage.Session, error) {
				return &storage.Session{
					Response: storage.SessionResponse{Code: 200, Delay: 7 * time.Second},
					Meta:     storage.SessionMeta{CreatedAt: fixedNow},
				}, nil
			},
			check: func(t *testing.T, resp *openapi.SessionOptionsResponse) {
				assert.Equal(t, uint16(7), resp.Response.Delay)
			},
		},

		"success/status code converted from uint16 to int": {
			giveReq: fixedSID,
			mockFn: func(_ context.Context, _ string) (*storage.Session, error) {
				return &storage.Session{
					Response: storage.SessionResponse{Code: 418},
					Meta:     storage.SessionMeta{CreatedAt: fixedNow},
				}, nil
			},
			check: func(t *testing.T, resp *openapi.SessionOptionsResponse) {
				assert.Equal(t, 418, resp.Response.StatusCode)
			},
		},

		"error/session not found returns openapi not found": {
			giveReq: fixedSID,
			mockFn: func(_ context.Context, _ string) (*storage.Session, error) {
				return nil, storage.ErrSessionNotFound
			},
			wantErr: true,
			checkErr: func(t *testing.T, err error) {
				assert.ErrorIs(t, err, openapi.ErrNotFound)
			},
		},

		"error/storage failure wrapped": {
			giveReq: fixedSID,
			mockFn: func(_ context.Context, _ string) (*storage.Session, error) {
				return nil, errors.New("storage unavailable")
			},
			wantErr: true,
			checkErr: func(t *testing.T, err error) {
				assert.ErrorContains(t, err, "get session")
				assert.ErrorContains(t, err, "storage unavailable")
			},
		},
	} {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			h := session_get.New(&mockStorage{getSessionFn: tt.mockFn})
			resp, err := h.Handle(t.Context(), tt.giveReq)

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

var _ storage.SessionStorage = (*mockStorage)(nil) // compile-time interface assertion

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
