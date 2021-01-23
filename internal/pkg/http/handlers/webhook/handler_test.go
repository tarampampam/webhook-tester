package webhook_test

import (
	"bytes"
	"context"
	"io"
	"net/http"
	"net/http/httptest"
	"runtime"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/tarampampam/webhook-tester/internal/pkg/http/handlers/webhook"

	"github.com/gorilla/mux"
	"github.com/stretchr/testify/assert"
	"github.com/tarampampam/webhook-tester/internal/pkg/broadcast"
	"github.com/tarampampam/webhook-tester/internal/pkg/config"
	"github.com/tarampampam/webhook-tester/internal/pkg/storage"
)

func BenchmarkHandler_ServeHTTP(b *testing.B) {
	b.ReportAllocs()

	s := storage.NewInMemoryStorage(time.Minute, 10)
	defer s.Close()

	var (
		req, _ = http.NewRequest(http.MethodPut, "http://test", http.NoBody)
		rr     = httptest.NewRecorder()
		h      = webhook.NewHandler(context.Background(), config.Config{
			IgnoreHeaderPrefixes: []string{"bar", "baz"},
		}, s, &broadcast.None{})
	)

	sessionUUID, _ := s.CreateSession("foo", 202, "foo/bar", 0)

	req = mux.SetURLVars(req, map[string]string{"sessionUUID": sessionUUID, "statusCode": "222"})

	req.Header.Set("foo", "blah")
	req.Header.Set("X-Forwarded-For", "4.4.4.4")
	req.Header.Set("X-Forwarded-For1", "4.4.4.4")
	req.Header.Set("X-Forwarded-For2", "4.4.4.4")
	req.Header.Set("X-Forwarded-For3", "4.4.4.4")
	req.Header.Set("X-Forwarded-For4", "4.4.4.4")
	req.Header.Set("X-Forwarded-For5", "4.4.4.4")

	for n := 0; n < b.N; n++ {
		h.ServeHTTP(rr, req)
	}
}

func TestHandler_ServeHTTPRequestErrors(t *testing.T) {
	var cases = []struct {
		name           string
		giveBody       io.Reader
		giveReqVars    func(s storage.Storage) map[string]string
		wantStatusCode int
		wantJSON       string
	}{
		{
			name:           "without registered session UUID",
			giveReqVars:    nil,
			wantStatusCode: http.StatusInternalServerError,
			wantJSON:       `{"code":500,"success":false,"message":"cannot extract session UUID"}`,
		},
		{
			name: "session was not found",
			giveReqVars: func(s storage.Storage) map[string]string {
				return map[string]string{"sessionUUID": "aa-bb-cc-dd"}
			},
			wantStatusCode: http.StatusNotFound,
			wantJSON:       `{"code":404,"success":false,"message":"session with UUID aa-bb-cc-dd was not found"}`,
		},
		{
			name: "too large body request",
			giveReqVars: func(s storage.Storage) map[string]string {
				sUUID, err := s.CreateSession("", 202, "", 0)
				assert.NoError(t, err)

				return map[string]string{"sessionUUID": sUUID}
			},
			giveBody:       bytes.NewBuffer([]byte(strings.Repeat("x", 65))),
			wantStatusCode: http.StatusInternalServerError,
			wantJSON:       `{"code":500,"success":false,"message":"request body is too large (current: 65, maximal: 64)"}`, //nolint:lll
		},
	}

	for _, tt := range cases {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			s := storage.NewInMemoryStorage(time.Minute, 10)
			defer s.Close()

			var (
				req, _  = http.NewRequest(http.MethodPost, "http://test", tt.giveBody)
				rr      = httptest.NewRecorder()
				br      = &broadcast.None{}
				handler = webhook.NewHandler(context.Background(), config.Config{
					MaxRequestBodySize: 64,
				}, s, br)
			)

			if tt.giveReqVars != nil {
				req = mux.SetURLVars(req, tt.giveReqVars(s))
			}

			handler.ServeHTTP(rr, req)

			assert.Equal(t, tt.wantStatusCode, rr.Code)
			assert.JSONEq(t, tt.wantJSON, rr.Body.String())
		})
	}
}

func TestHandler_ServeHTTPSuccess(t *testing.T) {
	s := storage.NewInMemoryStorage(time.Minute, 10)
	defer s.Close()

	var (
		req, _  = http.NewRequest(http.MethodPost, "http://test", bytes.NewBuffer([]byte("foo=bar")))
		rr      = httptest.NewRecorder()
		br      = &broadcast.None{}
		handler = webhook.NewHandler(context.Background(), config.Config{
			IgnoreHeaderPrefixes: []string{"x-bAr-", "Baz"},
		}, s, br)
	)

	var (
		brChannel string
		brEvent   broadcast.Event
		brCount   int
		brMutex   sync.Mutex
	)

	br.OnPublish(func(ch string, e broadcast.Event) {
		brMutex.Lock()
		brChannel, brEvent = ch, e
		brCount++
		brMutex.Unlock()
	})

	req.Header.Set("x-bar-foo", "baz") // must be ignored
	req.Header.Set("bAZ", "foo")       // must be ignored
	req.Header.Set("foo", "blah")
	req.Header.Set("X-Forwarded-For", "4.4.4.4")
	req.Header.Set("X-Real-IP", "3.3.3.3")
	req.Header.Set("cf-connecting-ip", "2.2.2.2, 2.1.1.2")

	sessionUUID, err := s.CreateSession("foo", 202, "foo/bar", 0)
	assert.NoError(t, err)

	req = mux.SetURLVars(req, map[string]string{"sessionUUID": sessionUUID})

	handler.ServeHTTP(rr, req)

	runtime.Gosched()
	<-time.After(time.Millisecond) // FIXME goroutine must be done

	assert.Equal(t, 202, rr.Code)
	assert.Equal(t, "foo", rr.Body.String())
	assert.Equal(t, "foo/bar", rr.Header().Get("Content-Type"))

	brMutex.Lock()
	assert.Equal(t, 1, brCount)
	assert.Equal(t, sessionUUID, brChannel)
	assert.Equal(t, "request-registered", brEvent.Name())
	brMutex.Unlock()

	requests, err := s.GetAllRequests(sessionUUID)
	assert.NoError(t, err)

	assert.Equal(t, http.MethodPost, requests[0].Method())
	assert.Equal(t, "foo=bar", requests[0].Content())
	assert.Equal(t, map[string]string{
		"Foo":              "blah",
		"X-Forwarded-For":  "4.4.4.4",
		"X-Real-Ip":        "3.3.3.3",
		"Cf-Connecting-Ip": "2.2.2.2, 2.1.1.2",
	}, requests[0].Headers())
	assert.Equal(t, "2.2.2.2", requests[0].ClientAddr())
}

func TestHandler_ServeHTTPSuccessCustomCode(t *testing.T) {
	s := storage.NewInMemoryStorage(time.Minute, 10)
	defer s.Close()

	var (
		req, _  = http.NewRequest(http.MethodPut, "http://test", http.NoBody)
		rr      = httptest.NewRecorder()
		handler = webhook.NewHandler(context.Background(), config.Config{}, s, &broadcast.None{})
	)

	sessionUUID, err := s.CreateSession("foo", 202, "foo/bar", 0)
	assert.NoError(t, err)

	req = mux.SetURLVars(req, map[string]string{"sessionUUID": sessionUUID, "statusCode": "222"})

	handler.ServeHTTP(rr, req)

	assert.Equal(t, 222, rr.Code)
	assert.Equal(t, "foo", rr.Body.String())
	assert.Equal(t, "foo/bar", rr.Header().Get("Content-Type"))

	requests, err := s.GetAllRequests(sessionUUID)
	assert.NoError(t, err)

	assert.Equal(t, http.MethodPut, requests[0].Method())
	assert.Equal(t, "", requests[0].Content())
	assert.Empty(t, requests[0].Headers())
}

func TestHandler_ServeHTTPSuccessWrongCustomCode(t *testing.T) {
	s := storage.NewInMemoryStorage(time.Minute, 10)
	defer s.Close()

	var (
		req, _  = http.NewRequest(http.MethodPut, "http://test", http.NoBody)
		rr      = httptest.NewRecorder()
		handler = webhook.NewHandler(context.Background(), config.Config{}, s, &broadcast.None{})
	)

	sessionUUID, err := s.CreateSession("foo", 203, "foo/bar", 0)
	assert.NoError(t, err)

	req = mux.SetURLVars(req, map[string]string{"sessionUUID": sessionUUID, "statusCode": "999"}) // wrong code

	handler.ServeHTTP(rr, req)

	assert.Equal(t, 203, rr.Code)
	assert.Equal(t, "foo", rr.Body.String())
	assert.Equal(t, "foo/bar", rr.Header().Get("Content-Type"))

	requests, err := s.GetAllRequests(sessionUUID)
	assert.NoError(t, err)

	assert.Equal(t, http.MethodPut, requests[0].Method())
	assert.Equal(t, "", requests[0].Content())
	assert.Empty(t, requests[0].Headers())
}

func TestHandler_ServeHTTPDelay(t *testing.T) {
	s := storage.NewInMemoryStorage(time.Minute, 10)
	defer s.Close()

	var (
		req, _  = http.NewRequest(http.MethodPut, "http://test", http.NoBody)
		rr      = httptest.NewRecorder()
		handler = webhook.NewHandler(context.Background(), config.Config{}, s, &broadcast.None{})
	)

	sessionUUID, err := s.CreateSession("foo", 203, "foo/bar", time.Millisecond*100)
	assert.NoError(t, err)

	req = mux.SetURLVars(req, map[string]string{"sessionUUID": sessionUUID})

	start := time.Now().UnixNano()

	handler.ServeHTTP(rr, req)

	end := time.Now().UnixNano()

	assert.InDelta(t, time.Millisecond*100, time.Duration(end-start), float64(time.Millisecond*5))

	assert.Equal(t, 203, rr.Code)
	assert.Equal(t, "foo", rr.Body.String())
	assert.Equal(t, "foo/bar", rr.Header().Get("Content-Type"))

	requests, err := s.GetAllRequests(sessionUUID)
	assert.NoError(t, err)

	assert.Equal(t, http.MethodPut, requests[0].Method())
	assert.Equal(t, "", requests[0].Content())
	assert.Empty(t, requests[0].Headers())
}

func TestHandler_ServeHTTPContextCancellation(t *testing.T) {
	s := storage.NewInMemoryStorage(time.Minute, 10)
	defer s.Close()

	ctx, cancel := context.WithCancel(context.Background())

	var (
		req, _  = http.NewRequest(http.MethodPut, "http://test", http.NoBody)
		rr      = httptest.NewRecorder()
		handler = webhook.NewHandler(ctx, config.Config{}, s, &broadcast.None{})
	)

	sessionUUID, err := s.CreateSession("foo", 203, "foo/bar", time.Hour)
	assert.NoError(t, err)

	req = mux.SetURLVars(req, map[string]string{"sessionUUID": sessionUUID})

	cancel()
	handler.ServeHTTP(rr, req)

	assert.Equal(t, http.StatusInternalServerError, rr.Code)
	assert.JSONEq(t, `{"code":500,"success":false,"message":"canceled"}`, rr.Body.String())
	assert.Equal(t, "application/json", rr.Header().Get("Content-Type"))

	requests, err := s.GetAllRequests(sessionUUID)
	assert.NoError(t, err)

	assert.Equal(t, http.MethodPut, requests[0].Method())
	assert.Equal(t, "", requests[0].Content())
	assert.Empty(t, requests[0].Headers())
}
