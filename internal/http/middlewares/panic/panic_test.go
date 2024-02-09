package panic_test

import (
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/kami-zh/go-capturer"
	"github.com/labstack/echo/v4"
	"github.com/stretchr/testify/assert"
	"go.uber.org/zap"

	panicMiddlewares "gh.tarampamp.am/webhook-tester/internal/http/middlewares/panic"
)

func TestMiddleware(t *testing.T) {
	e := echo.New()

	for name, tt := range map[string]struct {
		giveHandler echo.HandlerFunc
		giveRequest func() *http.Request
		checkResult func(t *testing.T, in map[string]interface{}, rr *httptest.ResponseRecorder)
	}{
		"panic with error": {
			giveHandler: func(c echo.Context) error {
				panic(errors.New("foo error"))
			},
			giveRequest: func() *http.Request {
				rq, _ := http.NewRequest(http.MethodGet, "http://testing/foo/bar", http.NoBody)

				return rq
			},
			checkResult: func(t *testing.T, in map[string]interface{}, rr *httptest.ResponseRecorder) {
				// check log entry
				assert.Equal(t, "foo error", in["error"])
				assert.Contains(t, in["stacktrace"], "/panic.go:")
				assert.Contains(t, in["stacktrace"], ".TestMiddleware")

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
		"panic with string": {
			giveHandler: func(c echo.Context) error {
				panic("bar error")
			},
			giveRequest: func() *http.Request {
				rq, _ := http.NewRequest(http.MethodGet, "http://testing/foo/bar", http.NoBody)

				return rq
			},
			checkResult: func(t *testing.T, in map[string]interface{}, rr *httptest.ResponseRecorder) {
				assert.Equal(t, "bar error", in["error"])
			},
		},
	} {
		tt := tt

		t.Run(name, func(t *testing.T) {
			var (
				rr = httptest.NewRecorder()
				c  = e.NewContext(tt.giveRequest(), rr)
			)

			output := capturer.CaptureStderr(func() {
				log, err := zap.NewProduction()
				assert.NoError(t, err)

				err = panicMiddlewares.New(log)(tt.giveHandler)(c)
				assert.NoError(t, err)
			})

			var asJSON map[string]interface{}

			assert.NoError(t, json.Unmarshal([]byte(output), &asJSON), "logger output must be valid JSON")

			tt.checkResult(t, asJSON, rr)
		})
	}
}
