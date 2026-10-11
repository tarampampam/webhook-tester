package session_create_test

import (
	"context"
	"encoding/base64"
	"errors"
	"net/http"
	"testing"
	"time"

	"gh.tarampamp.am/webhook-tester/v3/internal/httpserver/handlers/session_create"
	"gh.tarampamp.am/webhook-tester/v3/internal/httpserver/openapi"
	"gh.tarampamp.am/webhook-tester/v3/internal/storage"
	"gh.tarampamp.am/webhook-tester/v3/internal/testutil/assert"
)

func TestHandler_Handle(t *testing.T) {
	t.Parallel()

	var (
		fixedNow  = time.Date(2024, 6, 15, 12, 0, 0, 0, time.UTC)
		fixedMeta = &storage.SessionMeta{CreatedAt: fixedNow}
	)

	for name, tt := range map[string]struct {
		giveReq  openapi.CreateSessionRequest
		giveTTL  time.Duration
		mockFn   func(context.Context, string, storage.SessionResponse, time.Duration) (*storage.SessionMeta, error)
		wantErr  bool
		checkErr func(*testing.T, error)
		check    func(*testing.T, *openapi.CreateSessionResponse)
	}{
		"success/minimal request": {
			giveReq: openapi.CreateSessionRequest{
				StatusCode:         http.StatusOK,
				Headers:            []openapi.HttpHeader{},
				ResponseBodyBase64: "",
			},
			giveTTL: storage.NoExpiration,
			mockFn: func(_ context.Context, sID string, _ storage.SessionResponse, _ time.Duration) (*storage.SessionMeta, error) {
				assert.True(t, openapi.IsValidUUID(sID))

				return fixedMeta, nil
			},
			check: func(t *testing.T, resp *openapi.CreateSessionResponse) {
				assert.True(t, openapi.IsValidUUID(resp.Uuid))
				assert.Equal(t, fixedNow.UnixMilli(), resp.CreatedAtUnixMilli)
			},
		},

		"success/body base64 decoded for storage": {
			giveReq: openapi.CreateSessionRequest{
				StatusCode:         http.StatusOK,
				Headers:            []openapi.HttpHeader{},
				ResponseBodyBase64: base64.StdEncoding.EncodeToString([]byte("hello body")),
			},
			giveTTL: storage.NoExpiration,
			mockFn: func(_ context.Context, _ string, r storage.SessionResponse, _ time.Duration) (*storage.SessionMeta, error) {
				assert.DeepEqual(t, []byte("hello body"), r.Body)

				return fixedMeta, nil
			},
		},

		"success/headers converted to storage format": {
			giveReq: openapi.CreateSessionRequest{
				StatusCode: 201,
				Headers: []openapi.HttpHeader{
					{Name: "Content-Type", Value: "application/json"},
					{Name: "X-Custom", Value: "value"},
				},
				ResponseBodyBase64: "",
			},
			giveTTL: storage.NoExpiration,
			mockFn: func(_ context.Context, _ string, r storage.SessionResponse, _ time.Duration) (*storage.SessionMeta, error) {
				assert.DeepEqual(t, []storage.ResponseHeader{
					{Name: "Content-Type", Value: "application/json"},
					{Name: "X-Custom", Value: "value"},
				}, r.Headers)

				return fixedMeta, nil
			},
		},

		"success/delay in seconds converted to duration for storage": {
			giveReq: openapi.CreateSessionRequest{
				StatusCode:         200,
				Headers:            []openapi.HttpHeader{},
				ResponseBodyBase64: "",
				Delay:              7,
			},
			giveTTL: storage.NoExpiration,
			mockFn: func(_ context.Context, _ string, r storage.SessionResponse, _ time.Duration) (*storage.SessionMeta, error) {
				assert.Equal(t, 7*time.Second, r.Delay)

				return fixedMeta, nil
			},
		},

		"success/status code passed to storage": {
			giveReq: openapi.CreateSessionRequest{
				StatusCode:         418,
				Headers:            []openapi.HttpHeader{},
				ResponseBodyBase64: "",
			},
			giveTTL: storage.NoExpiration,
			mockFn: func(_ context.Context, _ string, r storage.SessionResponse, _ time.Duration) (*storage.SessionMeta, error) {
				assert.Equal(t, uint16(418), r.Code)

				return fixedMeta, nil
			},
		},

		"success/ttl from constructor forwarded to storage": {
			giveReq: openapi.CreateSessionRequest{
				StatusCode:         200,
				Headers:            []openapi.HttpHeader{},
				ResponseBodyBase64: "",
			},
			giveTTL: 24 * time.Hour,
			mockFn: func(_ context.Context, _ string, _ storage.SessionResponse, ttl time.Duration) (*storage.SessionMeta, error) {
				assert.Equal(t, 24*time.Hour, ttl)

				return fixedMeta, nil
			},
		},

		"error/invalid base64 body": {
			giveReq: openapi.CreateSessionRequest{
				StatusCode:         200,
				Headers:            []openapi.HttpHeader{},
				ResponseBodyBase64: "not-valid-base64!!!",
			},
			giveTTL: time.Hour,
			mockFn:  nil, // storage must not be called
			wantErr: true,
			checkErr: func(t *testing.T, err error) {
				assert.ErrorContains(t, err, "decode response body")
			},
		},

		"error/storage failure wrapped in response": {
			giveReq: openapi.CreateSessionRequest{
				StatusCode:         200,
				Headers:            []openapi.HttpHeader{},
				ResponseBodyBase64: "",
			},
			giveTTL: time.Hour,
			mockFn: func(_ context.Context, _ string, _ storage.SessionResponse, _ time.Duration) (*storage.SessionMeta, error) {
				return nil, errors.New("storage unavailable")
			},
			wantErr: true,
			checkErr: func(t *testing.T, err error) {
				assert.ErrorContains(t, err, "create session")
				assert.ErrorContains(t, err, "storage unavailable")
			},
		},
	} {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			h := session_create.New(&mockStorage{newSessionFn: tt.mockFn}, tt.giveTTL)
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
	newSessionFn func(context.Context, string, storage.SessionResponse, time.Duration) (*storage.SessionMeta, error)
}

var _ storage.SessionStorage = (*mockStorage)(nil) // compile-time interface assertion

func (m *mockStorage) NewSession(
	ctx context.Context, sID string, resp storage.SessionResponse, ttl time.Duration,
) (*storage.SessionMeta, error) {
	if m.newSessionFn == nil {
		panic("mockStorage: unexpected NewSession call")
	}

	return m.newSessionFn(ctx, sID, resp, ttl)
}

func (m *mockStorage) GetSession(context.Context, string) (*storage.Session, error) {
	panic("mockStorage: GetSession not expected in this handler")
}

func (m *mockStorage) AddSessionTTL(context.Context, string, time.Duration) error {
	panic("mockStorage: AddSessionTTL not expected in this handler")
}

func (m *mockStorage) DeleteSession(context.Context, string) error {
	panic("mockStorage: DeleteSession not expected in this handler")
}
