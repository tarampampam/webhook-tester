package all

import (
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"
	"webhook-tester/storage"
	nullStorage "webhook-tester/storage/null"

	"github.com/gorilla/mux"
	"github.com/stretchr/testify/assert"
)

func TestJSONRPCHandler_ServeHTTP(t *testing.T) {
	t.Parallel()

	var cases = []struct {
		name        string
		giveReqVars map[string]string
		setUp       func(s *nullStorage.Storage)
		checkResult func(t *testing.T, rr *httptest.ResponseRecorder)
	}{
		{
			name:        "without registered session UUID",
			giveReqVars: nil,
			checkResult: func(t *testing.T, rr *httptest.ResponseRecorder) {
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
			checkResult: func(t *testing.T, rr *httptest.ResponseRecorder) {
				assert.Equal(t, http.StatusInternalServerError, rr.Code)
				assert.JSONEq(t,
					`{"code":500,"success":false,"message":"cannot get session data: foo"}`, rr.Body.String(),
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
			checkResult: func(t *testing.T, rr *httptest.ResponseRecorder) {
				assert.Equal(t, http.StatusNotFound, rr.Code)
				assert.JSONEq(t,
					`{"code":404,"success":false,"message":"session with UUID aa-bb-cc-dd was not found"}`, rr.Body.String(),
				)
			},
		},
		{
			name:        "success with one item (headers must be sorted)",
			giveReqVars: map[string]string{"sessionUUID": "aa-bb-cc-dd"},
			setUp: func(s *nullStorage.Storage) {
				s.Error = nil
				s.Boolean = true
				s.SessionData = &storage.SessionData{
					UUID: "aa-bb-cc-dd",
				}
				s.Requests = &[]storage.RequestData{
					{
						UUID: "11-22-33-44",
						Request: storage.Request{
							ClientAddr: "1.2.2.1",
							Method:     "PUT",
							Content:    "foobar",
							Headers:    map[string]string{"bbb": "foo", "aaa": "bar"},
							URI:        "http://example.com/foo",
						},
						CreatedAtUnix: 1,
					},
				}
			},
			checkResult: func(t *testing.T, rr *httptest.ResponseRecorder) {
				assert.Equal(t, http.StatusOK, rr.Code)
				assert.JSONEq(t, `[{
					"client_address":"1.2.2.1",
					"content":"foobar",
					"created_at_unix":1,
					"headers":[{"name": "aaa", "value": "bar"},{"name": "bbb", "value": "foo"}],
					"method":"PUT",
					"url":"http://example.com/foo",
					"uuid":"11-22-33-44"
				}]`, rr.Body.String(),
				)
			},
		},
		{
			name:        "success with many items (must be sorted)",
			giveReqVars: map[string]string{"sessionUUID": "aa-bb-cc-dd"},
			setUp: func(s *nullStorage.Storage) {
				s.Error = nil
				s.Boolean = true
				s.SessionData = &storage.SessionData{
					UUID: "aa-bb-cc-dd",
				}
				s.Requests = &[]storage.RequestData{
					{
						UUID:          "111",
						Request:       storage.Request{},
						CreatedAtUnix: 3,
					},
					{
						UUID:          "222",
						Request:       storage.Request{},
						CreatedAtUnix: 1,
					},
					{
						UUID:          "333",
						Request:       storage.Request{},
						CreatedAtUnix: 10,
					},
					{
						UUID:          "444",
						Request:       storage.Request{},
						CreatedAtUnix: 2,
					},
				}
			},
			checkResult: func(t *testing.T, rr *httptest.ResponseRecorder) {
				assert.Equal(t, http.StatusOK, rr.Code)
				assert.JSONEq(t, `[
					{
						"client_address":"",
						"content":"",
						"created_at_unix":10,
						"headers":[],
						"method":"",
						"url":"",
						"uuid":"333"
					},
					{
						"client_address":"",
						"content":"",
						"created_at_unix":3,
						"headers":[],
						"method":"",
						"url":"",
						"uuid":"111"
					},
					{
						"client_address":"",
						"content":"",
						"created_at_unix":2,
						"headers":[],
						"method":"",
						"url":"",
						"uuid":"444"
					},
					{
						"client_address":"",
						"content":"",
						"created_at_unix":1,
						"headers":[],
						"method":"",
						"url":"",
						"uuid":"222"
					}
				]`, rr.Body.String(),
				)
			},
		},
	}

	for _, tt := range cases {
		t.Run(tt.name, func(t *testing.T) {
			var (
				req, _  = http.NewRequest(http.MethodPost, "http://testing", nil)
				rr      = httptest.NewRecorder()
				s       = &nullStorage.Storage{}
				handler = NewHandler(s)
			)

			if tt.giveReqVars != nil {
				req = mux.SetURLVars(req, tt.giveReqVars)
			}

			if tt.setUp != nil {
				tt.setUp(s)
			}

			handler.ServeHTTP(rr, req)

			tt.checkResult(t, rr)
		})
	}
}
