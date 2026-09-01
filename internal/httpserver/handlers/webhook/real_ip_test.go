package webhook_test

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"gh.tarampamp.am/webhook-tester/v3/internal/httpserver/handlers/webhook"
	"gh.tarampamp.am/webhook-tester/v3/internal/logger"
	"gh.tarampamp.am/webhook-tester/v3/internal/storage"
	"gh.tarampamp.am/webhook-tester/v3/internal/testutil/assert"
)

// TestExtractRealIP verifies client IP extraction by inspecting CapturedRequest.ClientAddr.
func TestExtractRealIP(t *testing.T) {
	t.Parallel()

	for name, tt := range map[string]struct {
		giveRemoteAddr string
		giveHeaders    map[string]string
		wantClientAddr string
	}{
		"CF-Connecting-IP used":                      {giveRemoteAddr: "10.0.0.1:1234", giveHeaders: map[string]string{"CF-Connecting-IP": "1.2.3.4"}, wantClientAddr: "1.2.3.4"},
		"X-Real-IP used":                             {giveRemoteAddr: "10.0.0.1:1234", giveHeaders: map[string]string{"X-Real-IP": "9.10.11.12"}, wantClientAddr: "9.10.11.12"},
		"X-Forwarded-For single IP":                  {giveRemoteAddr: "10.0.0.1:1234", giveHeaders: map[string]string{"X-Forwarded-For": "20.30.40.50"}, wantClientAddr: "20.30.40.50"},
		"X-Forwarded-For takes leftmost":             {giveRemoteAddr: "10.0.0.1:1234", giveHeaders: map[string]string{"X-Forwarded-For": "1.1.1.1, 2.2.2.2, 3.3.3.3"}, wantClientAddr: "1.1.1.1"},
		"CF-Connecting-IP wins over X-Forwarded-For": {giveRemoteAddr: "10.0.0.1:1234", giveHeaders: map[string]string{"CF-Connecting-IP": "1.2.3.4", "X-Forwarded-For": "9.9.9.9"}, wantClientAddr: "1.2.3.4"},
		"RemoteAddr fallback with port":              {giveRemoteAddr: "192.168.1.100:5678", giveHeaders: map[string]string{}, wantClientAddr: "192.168.1.100"},
		"RemoteAddr fallback without port":           {giveRemoteAddr: "192.168.1.100", giveHeaders: map[string]string{}, wantClientAddr: "192.168.1.100"},
		"IPv6 RemoteAddr":                            {giveRemoteAddr: "[::1]:1234", giveHeaders: map[string]string{}, wantClientAddr: "::1"},
	} {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			var capturedAddr string

			s := &mockStorage{
				getSessionFn: func(_ context.Context, _ string) (*storage.Session, error) {
					return &storage.Session{Response: storage.SessionResponse{Code: http.StatusOK}}, nil
				},
				newRequestFn: func(_ context.Context, _, _ string, req storage.CapturedRequest) (*storage.RequestMeta, error) {
					capturedAddr = req.ClientAddr

					return &storage.RequestMeta{CreatedAt: time.Now()}, nil
				},
			}

			h := webhook.New(logger.NewNop(), s, newNopPublisher(), false, time.Hour, 0)

			r := httptest.NewRequest(http.MethodPost, "/"+testSID, nil)
			r.RemoteAddr = tt.giveRemoteAddr

			for k, v := range tt.giveHeaders {
				r.Header.Set(k, v)
			}

			w := httptest.NewRecorder()
			h.Handle(w, r)

			assert.Equal(t, http.StatusOK, w.Code)
			assert.Equal(t, tt.wantClientAddr, capturedAddr)
		})
	}
}
