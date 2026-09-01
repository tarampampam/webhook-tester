package middleware_test

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"gh.tarampamp.am/webhook-tester/v3/internal/httpserver/middleware"
	"gh.tarampamp.am/webhook-tester/v3/internal/testutil/assert"
)

func TestCORSPreflight(t *testing.T) {
	t.Parallel()

	for name, tt := range map[string]struct {
		giveMethod  string
		giveHeaders map[string]string
		wantStatus  int
		wantHeaders map[string]string
		wantNext    bool
	}{
		"browser preflight - method only": {
			giveMethod:  http.MethodOptions,
			giveHeaders: map[string]string{"Access-Control-Request-Method": "POST"},
			wantStatus:  http.StatusNoContent,
			wantHeaders: map[string]string{
				"Access-Control-Allow-Origin":  "*",
				"Access-Control-Allow-Methods": "POST",
				"Access-Control-Max-Age":       "86400",
			},
			wantNext: false,
		},
		"browser preflight - method and headers": {
			giveMethod: http.MethodOptions,
			giveHeaders: map[string]string{
				"Access-Control-Request-Method":  "POST",
				"Access-Control-Request-Headers": "Content-Type, Authorization",
			},
			wantStatus: http.StatusNoContent,
			wantHeaders: map[string]string{
				"Access-Control-Allow-Origin":  "*",
				"Access-Control-Allow-Methods": "POST",
				"Access-Control-Allow-Headers": "Content-Type, Authorization",
				"Access-Control-Max-Age":       "86400",
			},
			wantNext: false,
		},
		"user-initiated OPTIONS without ACRM passes through": {
			giveMethod:  http.MethodOptions,
			giveHeaders: map[string]string{},
			wantStatus:  http.StatusOK,
			wantNext:    true,
		},
		"GET passes through": {
			giveMethod:  http.MethodGet,
			giveHeaders: map[string]string{},
			wantStatus:  http.StatusOK,
			wantNext:    true,
		},
		"POST passes through": {
			giveMethod:  http.MethodPost,
			giveHeaders: map[string]string{},
			wantStatus:  http.StatusOK,
			wantNext:    true,
		},
	} {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			var nextCalled bool

			handler := middleware.CORSPreflight(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
				nextCalled = true

				w.WriteHeader(http.StatusOK)
			}))

			r := httptest.NewRequest(tt.giveMethod, "/", nil)
			for k, v := range tt.giveHeaders {
				r.Header.Set(k, v)
			}

			w := httptest.NewRecorder()
			handler.ServeHTTP(w, r)

			assert.Equal(t, tt.wantStatus, w.Code)
			assert.Equal(t, tt.wantNext, nextCalled)

			for k, v := range tt.wantHeaders {
				assert.Equal(t, v, w.Header().Get(k))
			}
		})
	}
}
