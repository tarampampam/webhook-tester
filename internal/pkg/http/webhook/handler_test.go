package webhook

import (
	"bytes"
	"io"
	"net/http"
	"net/http/httptest"
	"runtime"
	"sync"
	"testing"
	"time"

	"github.com/gorilla/mux"
	"github.com/stretchr/testify/assert"
	"github.com/tarampampam/webhook-tester/internal/pkg/broadcast"
	"github.com/tarampampam/webhook-tester/internal/pkg/settings"
	"github.com/tarampampam/webhook-tester/internal/pkg/storage"
)

func TestHandler_ServeHTTPRequestErrors(t *testing.T) {
	var cases = []struct {
		name           string
		giveBody       io.Reader
		giveReqVars    map[string]string
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
			name:           "session was not found",
			giveReqVars:    map[string]string{"sessionUUID": "aa-bb-cc-dd"},
			wantStatusCode: http.StatusNotFound,
			wantJSON:       `{"code":404,"success":false,"message":"session with UUID aa-bb-cc-dd was not found"}`,
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
				handler = NewHandler(&settings.AppSettings{}, s, br)
			)

			if tt.giveReqVars != nil {
				req = mux.SetURLVars(req, tt.giveReqVars)
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
		handler = NewHandler(&settings.AppSettings{}, s, br)
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

	req.Header.Set("x-bar", "baz")

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
	assert.Equal(t, map[string]string{"X-Bar": "baz"}, requests[0].Headers())
}
