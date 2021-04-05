package delete_test

import (
	"net/http"
	"net/http/httptest"
	"runtime"
	"testing"
	"time"

	"github.com/gorilla/mux"
	"github.com/stretchr/testify/assert"
	"github.com/tarampampam/webhook-tester/internal/pkg/http/handlers/api/session/requests/delete"
	"github.com/tarampampam/webhook-tester/internal/pkg/pubsub"
	"github.com/tarampampam/webhook-tester/internal/pkg/storage"
)

func TestHandler_ServeHTTPRequestErrors(t *testing.T) {
	var cases = []struct {
		name           string
		giveReqVars    map[string]string
		wantStatusCode int
		wantJSON       string
	}{
		{
			name:           "without request params",
			giveReqVars:    nil,
			wantStatusCode: http.StatusInternalServerError,
			wantJSON:       `{"code":500,"success":false,"message":"cannot extract session UUID"}`,
		},
		{
			name:           "without request uuid",
			giveReqVars:    map[string]string{"sessionUUID": "aa-bb-cc-dd"},
			wantStatusCode: http.StatusInternalServerError,
			wantJSON:       `{"code":500,"success":false,"message":"cannot extract request UUID"}`,
		},
		{
			name:           "request not found",
			giveReqVars:    map[string]string{"sessionUUID": "aa-bb-cc-dd", "requestUUID": "11-22-33-44"},
			wantStatusCode: http.StatusNotFound,
			wantJSON:       `{"code":404,"success":false,"message":"request with UUID 11-22-33-44 was not found"}`,
		},
	}

	for _, tt := range cases {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			s := storage.NewInMemoryStorage(time.Minute, 10)
			defer s.Close()

			var (
				req, _  = http.NewRequest(http.MethodPost, "http://test", nil)
				rr      = httptest.NewRecorder()
				ps      = pubsub.NewInMemory()
				handler = delete.NewHandler(s, ps)
			)

			defer func() { _ = ps.Close() }()

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
		req, _  = http.NewRequest(http.MethodPost, "http://test", http.NoBody)
		rr      = httptest.NewRecorder()
		ps      = pubsub.NewInMemory()
		handler = delete.NewHandler(s, ps)
	)

	defer func() { _ = ps.Close() }()

	// create session
	sessionUUID, err := s.CreateSession("foo", 202, "foo/bar", 0)
	assert.NoError(t, err)

	// create 2 requests for the session
	_, _ = s.CreateRequest(sessionUUID, "", "", "", "", nil)
	requestUUID, _ := s.CreateRequest(sessionUUID, "", "", "", "", nil)
	requests, err := s.GetAllRequests(sessionUUID)
	assert.NoError(t, err)
	assert.Len(t, requests, 2) // is not empty

	// subscribe for events
	eventsCh := make(chan pubsub.Event, 3)
	assert.NoError(t, ps.Subscribe(sessionUUID, eventsCh))

	req = mux.SetURLVars(req, map[string]string{"sessionUUID": sessionUUID, "requestUUID": requestUUID})

	handler.ServeHTTP(rr, req)

	runtime.Gosched()
	<-time.After(time.Millisecond) // FIXME goroutine must be done

	assert.JSONEq(t, `{"success":true}`, rr.Body.String())

	e := <-eventsCh
	assert.Equal(t, "request-deleted", e.Name())
	assert.Equal(t, requestUUID, string(e.Data()))

	requests, err = s.GetAllRequests(sessionUUID)
	assert.NoError(t, err)
	assert.Len(t, requests, 1) // but still contains 1 request!
}
