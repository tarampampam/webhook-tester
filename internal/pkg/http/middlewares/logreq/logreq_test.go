package logreq

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/kami-zh/go-capturer"
	"github.com/stretchr/testify/assert"
	"go.uber.org/zap"
)

func TestMiddleware(t *testing.T) {
	cases := []struct {
		name              string
		giveRequest       func() *http.Request
		giveHandler       http.Handler
		checkOutputFields func(t *testing.T, in map[string]interface{})
	}{
		{
			name: "basic usage",
			giveHandler: http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
				time.Sleep(time.Millisecond)
				w.WriteHeader(http.StatusUnsupportedMediaType)
			}),
			giveRequest: func() (req *http.Request) {
				req, _ = http.NewRequest(http.MethodGet, "http://unit/test/?foo=bar&baz", nil)
				req.RemoteAddr = "4.3.2.1:567"
				req.Header.Set("User-Agent", "Foo Useragent")

				return
			},
			checkOutputFields: func(t *testing.T, in map[string]interface{}) {
				assert.Equal(t, http.MethodGet, in["method"])
				assert.NotZero(t, in["duration_micro"])
				assert.Equal(t, "info", in["level"])
				assert.Contains(t, in["msg"], "processed")
				assert.Equal(t, "4.3.2.1", in["remote_addr"])
				assert.Equal(t, float64(http.StatusUnsupportedMediaType), in["status_code"])
				assert.Equal(t, "http://unit/test/?foo=bar&baz", in["url"])
				assert.Equal(t, "Foo Useragent", in["useragent"])
			},
		},
		{
			name: "IP from 'CF-Connecting-IP' header",
			giveHandler: http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
				w.WriteHeader(http.StatusOK)
			}),
			giveRequest: func() (req *http.Request) {
				req, _ = http.NewRequest(http.MethodGet, "http://testing", nil)
				req.RemoteAddr = "4.4.4.4:567"
				req.Header.Set("X-Forwarded-For", "10.0.0.1, 10.0.0.2, 10.0.0.3")
				req.Header.Set("X-Real-IP", "10.0.1.1")
				req.Header.Set("CF-Connecting-IP", "10.1.1.1")

				return
			},
			checkOutputFields: func(t *testing.T, in map[string]interface{}) {
				assert.Equal(t, "10.1.1.1", in["remote_addr"])
			},
		},
		{
			name: "IP from 'X-Real-IP' header",
			giveHandler: http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				w.WriteHeader(http.StatusOK)
			}),
			giveRequest: func() (req *http.Request) {
				req, _ = http.NewRequest(http.MethodGet, "http://testing", nil)
				req.RemoteAddr = "8.8.8.8:567"
				req.Header.Set("X-Forwarded-For", "10.0.0.1, 10.0.0.2, 10.0.0.3")
				req.Header.Set("X-Real-IP", "10.0.1.1")

				return
			},
			checkOutputFields: func(t *testing.T, in map[string]interface{}) {
				assert.Equal(t, "10.0.1.1", in["remote_addr"])
			},
		},
		{
			name: "IP from 'X-Forwarded-For' header",
			giveHandler: http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				w.WriteHeader(http.StatusOK)
			}),
			giveRequest: func() (req *http.Request) {
				req, _ = http.NewRequest(http.MethodGet, "http://testing", nil)
				req.RemoteAddr = "1.2.3.4:567"
				req.Header.Set("X-Forwarded-For", "10.0.0.1, 10.0.0.2, 10.0.0.3")

				return
			},
			checkOutputFields: func(t *testing.T, in map[string]interface{}) {
				assert.Equal(t, "10.0.0.1", in["remote_addr"])
			},
		},
	}

	for _, tt := range cases {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			var rr = httptest.NewRecorder()

			output := capturer.CaptureStderr(func() {
				log, err := zap.NewProduction()
				assert.NoError(t, err)

				New(log).Middleware(tt.giveHandler).ServeHTTP(rr, tt.giveRequest())
			})

			var asJSON map[string]interface{}
			assert.NoError(t, json.Unmarshal([]byte(output), &asJSON), "logger output must be valid JSON")

			tt.checkOutputFields(t, asJSON)
		})
	}
}
