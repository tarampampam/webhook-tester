package logreq_test

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/kami-zh/go-capturer"
	"github.com/labstack/echo/v4"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"

	"gh.tarampamp.am/webhook-tester/internal/http/middlewares/logreq"
)

func TestNew(t *testing.T) {
	e := echo.New()

	for name, tt := range map[string]struct {
		giveRequest           func() *http.Request
		giveHandler           echo.HandlerFunc
		giveDbgRoutesPrefixes []string
		wantOutput            bool
		checkOutputFields     func(t *testing.T, in map[string]interface{})
	}{
		"basic usage": {
			giveHandler: func(c echo.Context) error {
				time.Sleep(time.Millisecond)

				return c.NoContent(http.StatusUnsupportedMediaType)
			},
			giveRequest: func() (req *http.Request) {
				req, _ = http.NewRequest(http.MethodGet, "http://unit/test/?foo=bar&baz", http.NoBody)
				req.RemoteAddr = "4.3.2.1:567"
				req.Header.Set("User-Agent", "Foo Useragent")

				return
			},
			wantOutput: true,
			checkOutputFields: func(t *testing.T, in map[string]interface{}) {
				assert.Equal(t, http.MethodGet, in["method"])
				assert.NotZero(t, in["duration"])
				assert.Equal(t, "info", in["level"])
				assert.Contains(t, in["msg"], "processed")
				assert.Equal(t, "4.3.2.1", in["remote addr"])
				assert.Equal(t, float64(http.StatusUnsupportedMediaType), in["status code"])
				assert.Equal(t, "http://unit/test/?foo=bar&baz", in["uri"])
				assert.Equal(t, "Foo Useragent", in["useragent"])
			},
		},
		"IP from 'X-Forwarded-For' header": {
			giveHandler: func(c echo.Context) error {
				return c.NoContent(http.StatusOK)
			},
			giveRequest: func() (req *http.Request) {
				req, _ = http.NewRequest(http.MethodGet, "http://testing", http.NoBody)
				req.RemoteAddr = "1.2.3.4:567"
				req.Header.Set("X-Forwarded-For", "10.0.0.1, 10.0.0.2, 10.0.0.3")

				return
			},
			wantOutput: true,
			checkOutputFields: func(t *testing.T, in map[string]interface{}) {
				assert.Equal(t, "10.0.0.1", in["remote addr"])
			},
		},
		"prefix skipped": {
			giveDbgRoutesPrefixes: []string{"/foo_bar"},
			giveHandler: func(c echo.Context) error {
				return c.NoContent(http.StatusOK)
			},
			giveRequest: func() (req *http.Request) {
				req, _ = http.NewRequest(http.MethodGet, "http://test/foo_bar/?foo=bar&baz", http.NoBody)
				req.Header.Set("User-Agent", "HealthCheck/Internal")

				return
			},
			wantOutput: false,
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
				require.NoError(t, err)

				err = logreq.New(log, tt.giveDbgRoutesPrefixes)(tt.giveHandler)(c)
				assert.NoError(t, err)
			})

			if tt.wantOutput {
				var asJSON map[string]interface{}
				assert.NoError(t, json.Unmarshal([]byte(output), &asJSON), "logger output must be valid JSON")

				tt.checkOutputFields(t, asJSON)
			} else {
				assert.Empty(t, output)
			}
		})
	}
}
