package webhook

import (
	"bytes"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/gorilla/mux"
	"github.com/stretchr/testify/assert"
	"github.com/tarampampam/webhook-tester/internal/pkg/broadcast"
	nullBroadcast "github.com/tarampampam/webhook-tester/internal/pkg/broadcast/null"
	"github.com/tarampampam/webhook-tester/internal/pkg/settings"
	"github.com/tarampampam/webhook-tester/internal/pkg/storage"
	nullStorage "github.com/tarampampam/webhook-tester/internal/pkg/storage/null"
)

func TestHandler_ServeHTTP(t *testing.T) {
	t.Parallel()

	var cases = []struct {
		name        string
		giveBody    io.Reader
		giveReqVars map[string]string
		setUp       func(s *nullStorage.Storage)
		checkResult func(t *testing.T, rr *httptest.ResponseRecorder, b *nullBroadcast.Broadcaster)
	}{
		{
			name:        "without registered session UUID",
			giveReqVars: nil,
			checkResult: func(t *testing.T, rr *httptest.ResponseRecorder, b *nullBroadcast.Broadcaster) {
				assert.Equal(t, http.StatusInternalServerError, rr.Code)
				assert.JSONEq(t,
					`{"code":500,"success":false,"message":"cannot extract session UUID"}`, rr.Body.String(),
				)
			},
		},
		{
			name:        "emulate storage error",
			giveReqVars: map[string]string{"sessionUUID": "aa-bb-cc-dd"},
			setUp: func(s *nullStorage.Storage) {
				s.Error = errors.New("foo")
			},
			checkResult: func(t *testing.T, rr *httptest.ResponseRecorder, b *nullBroadcast.Broadcaster) {
				assert.Equal(t, http.StatusInternalServerError, rr.Code)
				assert.JSONEq(t,
					`{"code":500,"success":false,"message":"cannot read session data from storage: foo"}`, rr.Body.String(),
				)
			},
		},
		{
			name:        "emulate 'session was not found'",
			giveReqVars: map[string]string{"sessionUUID": "aa-bb-cc-dd"},
			setUp: func(s *nullStorage.Storage) {
				s.Error = nil
				s.Boolean = false
			},
			checkResult: func(t *testing.T, rr *httptest.ResponseRecorder, b *nullBroadcast.Broadcaster) {
				assert.Equal(t, http.StatusNotFound, rr.Code)
				assert.JSONEq(t,
					`{"code":404,"success":false,"message":"session with UUID aa-bb-cc-dd was not found"}`, rr.Body.String(),
				)
			},
		},
		{
			name:        "nil body",
			giveReqVars: map[string]string{"sessionUUID": "aa-bb-cc-dd"},
			setUp: func(s *nullStorage.Storage) {
				s.Error = nil
				s.SessionData = &storage.SessionData{
					UUID: "aa-bb-cc-dd",
					WebHookResponse: storage.WebHookResponse{
						Content:     "foo",
						Code:        202,
						ContentType: "foo/bar",
					},
					CreatedAtUnix: 0,
				}
				s.RequestData = &storage.RequestData{UUID: "11-22-33-44"}
			},
			checkResult: func(t *testing.T, rr *httptest.ResponseRecorder, b *nullBroadcast.Broadcaster) {
				time.Sleep(time.Millisecond) // goroutine must be done

				assert.Equal(t, 202, rr.Code)
				assert.Equal(t, "foo", rr.Body.String())
				assert.Equal(t, "foo/bar", rr.Header().Get("Content-Type"))

				assert.Equal(t, "aa-bb-cc-dd", b.GetLastPublishedChannel())
				assert.Equal(t, broadcast.RequestRegistered, b.GetLastPublishedEventName())
				assert.Equal(t, "11-22-33-44", b.GetLastPublishedData())
			},
		},
		{
			name:        "string body",
			giveReqVars: map[string]string{"sessionUUID": "aa-bb-cc-dd"},
			giveBody:    bytes.NewBuffer([]byte(`foo=bar`)),
			setUp: func(s *nullStorage.Storage) {
				s.Error = nil
				s.SessionData = &storage.SessionData{
					UUID: "aa-bb-cc-dd",
					WebHookResponse: storage.WebHookResponse{
						Content: "foo",
						Code:    202,
					},
					CreatedAtUnix: 0,
				}
				s.RequestData = &storage.RequestData{UUID: "11-22-33-44"}
			},
			checkResult: func(t *testing.T, rr *httptest.ResponseRecorder, b *nullBroadcast.Broadcaster) {
				time.Sleep(time.Millisecond) // goroutine must be done

				assert.Equal(t, 202, rr.Code)
				assert.Equal(t, "foo", rr.Body.String())

				assert.Equal(t, "aa-bb-cc-dd", b.GetLastPublishedChannel())
				assert.Equal(t, broadcast.RequestRegistered, b.GetLastPublishedEventName())
				assert.Equal(t, "11-22-33-44", b.GetLastPublishedData())
			},
		},
	}

	for _, tt := range cases {
		t.Run(tt.name, func(t *testing.T) {
			var (
				req, _  = http.NewRequest(http.MethodPost, "http://testing", tt.giveBody)
				rr      = httptest.NewRecorder()
				s       = &nullStorage.Storage{}
				br      = &nullBroadcast.Broadcaster{}
				handler = NewHandler(&settings.AppSettings{}, s, br)
			)

			if tt.giveReqVars != nil {
				req = mux.SetURLVars(req, tt.giveReqVars)
			}

			if tt.setUp != nil {
				tt.setUp(s)
			}

			handler.ServeHTTP(rr, req)

			tt.checkResult(t, rr, br)
		})
	}
}
