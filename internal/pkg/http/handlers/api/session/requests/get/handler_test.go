package get_test

import (
	"net/http"
	"net/http/httptest"
	"strconv"
	"testing"
	"time"

	"github.com/gorilla/mux"
	"github.com/stretchr/testify/assert"
	"github.com/tarampampam/webhook-tester/internal/pkg/http/handlers/api/session/requests/get"
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
			name:           "without registered session UUID",
			giveReqVars:    nil,
			wantStatusCode: http.StatusInternalServerError,
			wantJSON:       `{"code":500,"success":false,"message":"cannot extract session UUID"}`,
		},
		{
			name:           "without registered request UUID",
			giveReqVars:    map[string]string{"sessionUUID": "aa-bb-cc-dd"},
			wantStatusCode: http.StatusInternalServerError,
			wantJSON:       `{"code":500,"success":false,"message":"cannot extract request UUID"}`,
		},
		{
			name:           "unknown session",
			giveReqVars:    map[string]string{"sessionUUID": "aa-bb-cc-dd", "requestUUID": "dd-cc-bb-aa"},
			wantStatusCode: http.StatusNotFound,
			wantJSON:       `{"code":404,"success":false,"message":"request with UUID dd-cc-bb-aa was not found"}`,
		},
	}

	for _, tt := range cases {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			s := storage.NewInMemoryStorage(time.Minute, 1)
			defer s.Close()

			var (
				req, _  = http.NewRequest(http.MethodPost, "http://testing", nil)
				rr      = httptest.NewRecorder()
				handler = get.NewHandler(s)
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

func TestHandler_RequestReading(t *testing.T) {
	s := storage.NewInMemoryStorage(time.Minute, 10)
	defer s.Close()

	var (
		req, _  = http.NewRequest(http.MethodGet, "http://test", http.NoBody)
		rr      = httptest.NewRecorder()
		handler = get.NewHandler(s)
	)

	// create session
	sessionUUID, err := s.CreateSession("foo", 202, "foo/bar", 0)
	assert.NoError(t, err)

	// create ONE request for the session
	requestUUID, err := s.CreateRequest(
		sessionUUID,
		"1.2.2.1",
		"PUT",
		"foobar",
		"http://example.com/foo",
		map[string]string{"bbb": "foo", "aaa": "bar"},
	)
	assert.NoError(t, err)

	request, _ := s.GetRequest(sessionUUID, requestUUID)

	req = mux.SetURLVars(req, map[string]string{"sessionUUID": sessionUUID, "requestUUID": request.UUID()})

	handler.ServeHTTP(rr, req)

	assert.JSONEq(t, `{
		"client_address":"1.2.2.1",
		"content":"foobar",
		"created_at_unix":`+strconv.FormatInt(request.CreatedAt().Unix(), 10)+`,
		"headers":[{"name": "aaa", "value": "bar"},{"name": "bbb", "value": "foo"}],
		"method":"PUT",
		"url":"http://example.com/foo",
		"uuid":"`+request.UUID()+`"
	}`, rr.Body.String())
}
