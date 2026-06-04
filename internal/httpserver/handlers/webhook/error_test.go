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

// TestRenderErrorMessage_FormatSelection exercises Accept/Content-Type content negotiation
// by triggering the 404 path (session not found, auto-create disabled).
func TestRenderErrorMessage_FormatSelection(t *testing.T) {
	t.Parallel()

	s := &mockStorage{
		getSessionFn: func(_ context.Context, _ string) (*storage.Session, error) {
			return nil, storage.ErrSessionNotFound
		},
	}

	h := webhook.New(logger.NewNop(), s, newNopPublisher(), false, time.Hour, 0)

	for name, tt := range map[string]struct {
		giveHeaders map[string]string
		wantCT      string
	}{
		"accept json":                       {giveHeaders: map[string]string{"Accept": "application/json"}, wantCT: "application/json; charset=utf-8"},
		"accept html":                       {giveHeaders: map[string]string{"Accept": "text/html"}, wantCT: "text/html; charset=utf-8"},
		"accept plain":                      {giveHeaders: map[string]string{"Accept": "text/plain"}, wantCT: "text/plain; charset=utf-8"},
		"accept multi-value picks first":    {giveHeaders: map[string]string{"Accept": "text/html, application/json"}, wantCT: "text/html; charset=utf-8"},
		"accept wildcard falls back to ct":  {giveHeaders: map[string]string{"Accept": "*/*", "Content-Type": "application/json"}, wantCT: "application/json; charset=utf-8"},
		"no accept ct html":                 {giveHeaders: map[string]string{"Content-Type": "text/html"}, wantCT: "text/html; charset=utf-8"},
		"no accept ct plain":                {giveHeaders: map[string]string{"Content-Type": "text/plain"}, wantCT: "text/plain; charset=utf-8"},
		"no headers defaults to plain text": {giveHeaders: map[string]string{}, wantCT: "text/plain; charset=utf-8"},
	} {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			r := httptest.NewRequest(http.MethodPost, "/"+testSID, nil)
			for k, v := range tt.giveHeaders {
				r.Header.Set(k, v)
			}

			w := httptest.NewRecorder()
			h.Handle(w, r)

			assert.Equal(t, http.StatusNotFound, w.Code)
			assert.Equal(t, tt.wantCT, w.Header().Get("Content-Type"))
		})
	}
}
