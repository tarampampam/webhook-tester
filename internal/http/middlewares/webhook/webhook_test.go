package webhook_test

import (
	"bytes"
	"context"
	"io"
	"net/http"
	"net/http/httptest"
	"runtime"
	"strings"
	"testing"
	"time"

	"github.com/labstack/echo/v4"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"gh.tarampamp.am/webhook-tester/internal/config"
	appHttp "gh.tarampamp.am/webhook-tester/internal/http"
	"gh.tarampamp.am/webhook-tester/internal/http/middlewares/webhook"
	"gh.tarampamp.am/webhook-tester/internal/pubsub"
	"gh.tarampamp.am/webhook-tester/internal/storage"
)

type fakeMetrics struct {
	c int
}

func (f *fakeMetrics) IncrementProcessedWebHooks() { f.c++ }

const testBaseUri = "http://test/"

func BenchmarkHandler(b *testing.B) {
	b.ReportAllocs()

	s := storage.NewInMemory(time.Minute, 10)
	defer s.Close()

	ps := pubsub.NewInMemory()
	defer ps.Close()

	var (
		e              = echo.New()
		sessionUUID, _ = s.CreateSession([]byte("foo"), 202, "foo/bar", 0)
		req, _         = http.NewRequest(http.MethodPut, testBaseUri+sessionUUID+"/222?foo=bar#anchor", http.NoBody)
		rr             = httptest.NewRecorder()
		c              = e.NewContext(req, rr)

		h = webhook.New(context.Background(), config.Config{
			IgnoreHeaderPrefixes: []string{"bar", "baz"},
		}, s, ps, &fakeMetrics{})
	)

	req.Header.Set("foo", "blah")
	req.Header.Set("X-Forwarded-For", "4.4.4.4")
	req.Header.Set("X-Forwarded-For1", "4.4.4.4")

	for n := 0; n < b.N; n++ {
		require.NoError(b, h(func(c echo.Context) error {
			return c.NoContent(http.StatusOK)
		})(c))
	}
}

func TestHandler_RequestErrors(t *testing.T) {
	s := storage.NewInMemory(time.Minute, 10)
	defer s.Close()

	sessionUUID, _ := s.CreateSession([]byte("foo"), 202, "foo/bar", 0)

	ps := pubsub.NewInMemory()
	defer ps.Close()

	for name, tt := range map[string]struct {
		giveBody       io.Reader
		giveUrl        string
		wantStatusCode int
		wantSubstring  []string
	}{
		"with broken session UUID format (middleware should be skipped)": {
			giveUrl:        "http://test/XXXbbab9-c197-4dd3-bc3f-3cb625382ZZZ",
			wantStatusCode: http.StatusOK,
		},
		"session was not found": {
			giveUrl:        "http://test/9b6bbab9-c197-4dd3-bc3f-3cb6253820c7/222?foo=bar",
			wantStatusCode: http.StatusNotFound,
			wantSubstring:  []string{"session with UUID 9b6bbab9-c197-4dd3-bc3f-3cb6253820c7 was not found"},
		},
		"too large body request": {
			giveUrl:        testBaseUri + sessionUUID + "/222?foo=bar#anchor",
			giveBody:       bytes.NewBuffer([]byte(strings.Repeat("x", 65))),
			wantStatusCode: http.StatusInternalServerError,
			wantSubstring:  []string{"Request body is too large"},
		},
	} {
		tt := tt

		t.Run(name, func(t *testing.T) {
			var (
				e      = echo.New()
				req, _ = http.NewRequest(http.MethodPut, tt.giveUrl, tt.giveBody)
				rr     = httptest.NewRecorder()
				c      = e.NewContext(req, rr)

				h = webhook.New(context.Background(), config.Config{
					MaxRequestBodySize: 64,
				}, s, ps, &fakeMetrics{})
			)

			require.NoError(t, h(func(c echo.Context) error {
				return c.NoContent(http.StatusOK)
			})(c))

			assert.Equal(t, tt.wantStatusCode, rr.Code)

			for i := 0; i < len(tt.wantSubstring); i++ {
				assert.Contains(t, rr.Body.String(), tt.wantSubstring[i])
			}
		})
	}
}

func TestHandler_Success(t *testing.T) {
	s := storage.NewInMemory(time.Minute, 10)
	defer s.Close()

	ps := pubsub.NewInMemory()
	defer ps.Close()

	sessionUUID, err := s.CreateSession([]byte("foo"), 202, "foo/bar", 0)
	require.NoError(t, err)

	var e = echo.New()
	e.IPExtractor = appHttp.NewIPExtractor() // just as an additional "feature" test

	var (
		req, _ = http.NewRequest(http.MethodPost, testBaseUri+sessionUUID, bytes.NewBuffer([]byte("foo=bar")))
		rr     = httptest.NewRecorder()
		m      = fakeMetrics{}
		c      = e.NewContext(req, rr)

		h = webhook.New(context.Background(), config.Config{
			IgnoreHeaderPrefixes: []string{"x-bAr-", "Baz"},
		}, s, ps, &m)
	)

	req.Header.Set("x-bar-foo", "baz") // should be ignored
	req.Header.Set("bAZ", "foo")       // should be ignored
	req.Header.Set("foo", "blah")
	req.Header.Set("X-Forwarded-For", "4.4.4.4")
	req.Header.Set("X-Real-IP", "3.3.3.3")
	req.Header.Set("cf-connecting-ip", "2.2.2.2, 2.1.1.2")

	// subscribe for events
	eventsCh := make(chan pubsub.Event, 3)
	assert.NoError(t, ps.Subscribe(sessionUUID, eventsCh))

	require.NoError(t, h(func(c echo.Context) error {
		return c.NoContent(http.StatusOK)
	})(c))

	runtime.Gosched()
	<-time.After(time.Millisecond) // goroutine must be done

	assert.Equal(t, 202, rr.Code)
	assert.Equal(t, "foo", rr.Body.String())
	assert.Equal(t, "foo/bar", rr.Header().Get("Content-Type"))

	requests, err := s.GetAllRequests(sessionUUID)
	assert.NoError(t, err)

	event := <-eventsCh
	assert.Equal(t, "request-registered", event.Name())
	assert.Equal(t, requests[0].UUID(), string(event.Data()))

	assert.Equal(t, 1, m.c)

	assert.Equal(t, http.MethodPost, requests[0].Method())
	assert.Equal(t, []byte("foo=bar"), requests[0].Content())
	assert.Equal(t, map[string]string{
		"Foo":              "blah",
		"X-Forwarded-For":  "4.4.4.4",
		"X-Real-Ip":        "3.3.3.3",
		"Cf-Connecting-Ip": "2.2.2.2, 2.1.1.2",
	}, requests[0].Headers())
	assert.Equal(t, "2.2.2.2", requests[0].ClientAddr())
}

func TestHandler_SuccessCustomCode(t *testing.T) {
	s := storage.NewInMemory(time.Minute, 10)
	defer s.Close()

	ps := pubsub.NewInMemory()
	defer ps.Close()

	sessionUUID, err := s.CreateSession([]byte("foo"), 202, "foo/bar", 0)
	require.NoError(t, err)

	var (
		req, _ = http.NewRequest(http.MethodPut, testBaseUri+sessionUUID+"/222", http.NoBody)
		rr     = httptest.NewRecorder()
		e      = echo.New()
		c      = e.NewContext(req, rr)

		handler = webhook.New(context.Background(), config.Config{}, s, ps, &fakeMetrics{})
	)

	require.NoError(t, handler(func(c echo.Context) error {
		return c.NoContent(http.StatusOK)
	})(c))

	assert.Equal(t, 222, rr.Code)
	assert.Equal(t, "foo", rr.Body.String())
	assert.Equal(t, "foo/bar", rr.Header().Get("Content-Type"))

	requests, err := s.GetAllRequests(sessionUUID)
	assert.NoError(t, err)

	assert.Equal(t, http.MethodPut, requests[0].Method())
	assert.Equal(t, []byte(""), requests[0].Content())
	assert.Empty(t, requests[0].Headers())
}

func TestHandler_SuccessWrongCustomCode(t *testing.T) {
	s := storage.NewInMemory(time.Minute, 10)
	defer s.Close()

	ps := pubsub.NewInMemory()
	defer ps.Close()

	sessionUUID, err := s.CreateSession([]byte("foo"), 234, "foo/bar", 0)
	require.NoError(t, err)

	var (
		req, _ = http.NewRequest(http.MethodPut, testBaseUri+sessionUUID+"/999", http.NoBody)
		rr     = httptest.NewRecorder()
		e      = echo.New()
		c      = e.NewContext(req, rr)

		handler = webhook.New(context.Background(), config.Config{}, s, ps, &fakeMetrics{})
	)

	require.NoError(t, handler(func(c echo.Context) error {
		return c.NoContent(http.StatusOK)
	})(c))

	assert.Equal(t, 234, rr.Code)
	assert.Equal(t, "foo", rr.Body.String())
	assert.Equal(t, "foo/bar", rr.Header().Get("Content-Type"))

	requests, err := s.GetAllRequests(sessionUUID)
	assert.NoError(t, err)

	assert.Equal(t, http.MethodPut, requests[0].Method())
	assert.Equal(t, []byte(""), requests[0].Content())
	assert.Empty(t, requests[0].Headers())
}

func TestHandler_ServeHTTPDelay(t *testing.T) {
	s := storage.NewInMemory(time.Minute, 10)
	defer s.Close()

	ps := pubsub.NewInMemory()
	defer ps.Close()

	sessionUUID, err := s.CreateSession([]byte("foo"), 203, "foo/bar", time.Millisecond*100)
	require.NoError(t, err)

	var (
		req, _ = http.NewRequest(http.MethodPut, testBaseUri+sessionUUID, http.NoBody)
		rr     = httptest.NewRecorder()
		e      = echo.New()
		c      = e.NewContext(req, rr)

		handler = webhook.New(context.Background(), config.Config{}, s, ps, &fakeMetrics{})
	)

	start := time.Now().UnixNano()

	require.NoError(t, handler(func(c echo.Context) error {
		return c.NoContent(http.StatusOK)
	})(c))

	end := time.Now().UnixNano()

	assert.InDelta(t, time.Millisecond*100, time.Duration(end-start), float64(time.Millisecond*10))

	assert.Equal(t, 203, rr.Code)
	assert.Equal(t, "foo", rr.Body.String())
	assert.Equal(t, "foo/bar", rr.Header().Get("Content-Type"))

	requests, err := s.GetAllRequests(sessionUUID)
	assert.NoError(t, err)

	assert.Equal(t, http.MethodPut, requests[0].Method())
	assert.Equal(t, []byte(""), requests[0].Content())
	assert.Empty(t, requests[0].Headers())
}
