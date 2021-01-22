package get

import (
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/tarampampam/webhook-tester/internal/pkg/config"
)

func TestHandler_ServeHTTP(t *testing.T) {
	t.Parallel()

	var cases = []struct {
		name        string
		setUp       func(cfg *config.Config)
		checkResult func(t *testing.T, rr *httptest.ResponseRecorder)
	}{
		{
			name: "without registered session UUID",
			setUp: func(cfg *config.Config) {
				cfg.Pusher.Cluster = "foo"
				cfg.Pusher.Key = "bar"
				cfg.MaxRequests = 123
				cfg.SessionTTL = time.Second * 321
			},
			checkResult: func(t *testing.T, rr *httptest.ResponseRecorder) {
				assert.Equal(t, http.StatusOK, rr.Code)
				assert.JSONEq(t, `{
					"version": "0.0.0@undefined",
					"pusher": {"key":"bar", "cluster":"foo"},
					"limits": {"max_requests":123, "session_lifetime_sec":321}
				}`, rr.Body.String())
			},
		},
	}

	for _, tt := range cases {
		t.Run(tt.name, func(t *testing.T) {
			var (
				req, _ = http.NewRequest(http.MethodPost, "http://testing", nil)
				rr     = httptest.NewRecorder()
				cfg    = config.Config{}
			)

			if tt.setUp != nil {
				tt.setUp(&cfg)
			}

			handler := NewHandler(cfg)

			handler.ServeHTTP(rr, req)

			tt.checkResult(t, rr)
		})
	}
}
