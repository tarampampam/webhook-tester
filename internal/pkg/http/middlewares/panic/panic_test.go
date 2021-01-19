package panic

import (
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/kami-zh/go-capturer"
	"github.com/stretchr/testify/assert"
	"go.uber.org/zap"
)

func TestMiddleware(t *testing.T) {
	cases := []struct {
		name        string
		giveHandler http.Handler
		giveRequest func() *http.Request
		checkResult func(t *testing.T, in map[string]interface{}, rr *httptest.ResponseRecorder)
	}{
		{
			name: "panic with error",
			giveHandler: http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				panic(errors.New("foo error"))
			}),
			giveRequest: func() *http.Request {
				rq, _ := http.NewRequest(http.MethodGet, "http://testing/foo/bar", nil)

				return rq
			},
			checkResult: func(t *testing.T, in map[string]interface{}, rr *httptest.ResponseRecorder) {
				// check log entry
				assert.Equal(t, "foo error", in["error"])
				assert.Contains(t, in["stacktrace"], "/panic.go:")
				assert.Contains(t, in["stacktrace"], ".ServeHTTP")

				// check HTTP response
				wantJSON, err := json.Marshal(struct {
					Message string `json:"message"`
					Code    int    `json:"code"`
				}{
					Message: "Internal Server Error: foo error",
					Code:    http.StatusInternalServerError,
				})
				assert.NoError(t, err)

				assert.Equal(t, http.StatusInternalServerError, rr.Code)
				assert.JSONEq(t, string(wantJSON), rr.Body.String())
			},
		},
		{
			name: "panic with string",
			giveHandler: http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				panic("bar error")
			}),
			giveRequest: func() *http.Request {
				rq, _ := http.NewRequest(http.MethodGet, "http://testing/foo/bar", nil)

				return rq
			},
			checkResult: func(t *testing.T, in map[string]interface{}, rr *httptest.ResponseRecorder) {
				assert.Equal(t, "bar error", in["error"])
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

			tt.checkResult(t, asJSON, rr)
		})
	}
}
