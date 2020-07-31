package create

import (
	"bytes"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"webhook-tester/storage"
	nullStorage "webhook-tester/storage/null"

	"github.com/stretchr/testify/assert"
)

func TestJSONRPCHandler_ServeHTTP(t *testing.T) {
	t.Parallel()

	var cases = []struct {
		name        string
		giveBody    io.Reader
		setUp       func(s *nullStorage.Storage)
		checkResult func(t *testing.T, rr *httptest.ResponseRecorder)
	}{
		{
			name:     "nil body",
			giveBody: nil,
			checkResult: func(t *testing.T, rr *httptest.ResponseRecorder) {
				assert.Equal(t, http.StatusBadRequest, rr.Code)
				assert.JSONEq(t,
					`{"code":400,"success":false,"message":"empty request body"}`,
					rr.Body.String(),
				)
			},
		},
		{
			name:     "wrong json",
			giveBody: bytes.NewBuffer([]byte(`{json`)),
			checkResult: func(t *testing.T, rr *httptest.ResponseRecorder) {
				assert.Equal(t, http.StatusBadRequest, rr.Code)
				assert.JSONEq(t,
					`{"code":400,"success":false,"message":"cannot parse passed json"}`,
					rr.Body.String(),
				)
			},
		},
		{
			name: "wrong value in correct json struct (Unmarshal error)",
			giveBody: bytes.NewBuffer([]byte(`{
				"content_type":null,
				"status_code":null,
				"response_delay":-9999,
				"response_body":""
			}`)),
			checkResult: func(t *testing.T, rr *httptest.ResponseRecorder) {
				assert.Equal(t, http.StatusBadRequest, rr.Code)
				assert.JSONEq(t,
					`{"code":400,"success":false,"message":"cannot parse passed json"}`,
					rr.Body.String(),
				)
			},
		},

		{
			name: "wrong value in json (response_delay)",
			giveBody: bytes.NewBuffer([]byte(`{
				"content_type":null,
				"status_code":null,
				"response_delay":99,
				"response_body":""
			}`)),
			checkResult: func(t *testing.T, rr *httptest.ResponseRecorder) {
				assert.Equal(t, http.StatusBadRequest, rr.Code)
				assert.JSONEq(t,
					`{"code":400,"success":false,"message":"invalid value passed: delay is too much"}`,
					rr.Body.String(),
				)
			},
		},
		{
			name: "wrong value in json (status_code)",
			giveBody: bytes.NewBuffer([]byte(`{
				"content_type":null,
				"status_code":1,
				"response_delay":null,
				"response_body":""
			}`)),
			checkResult: func(t *testing.T, rr *httptest.ResponseRecorder) {
				assert.Equal(t, http.StatusBadRequest, rr.Code)
				assert.JSONEq(t,
					`{"code":400,"success":false,"message":"invalid value passed: wrong status code value"}`,
					rr.Body.String(),
				)
			},
		},
		{
			name: "wrong value in json (content_type)",
			giveBody: bytes.NewBuffer([]byte(`{
				"content_type":"` + strings.Repeat("x", 512) + `",
				"status_code":null,
				"response_delay":null,
				"response_body":""
			}`)),
			checkResult: func(t *testing.T, rr *httptest.ResponseRecorder) {
				assert.Equal(t, http.StatusBadRequest, rr.Code)
				assert.JSONEq(t,
					`{"code":400,"success":false,"message":"invalid value passed: content-type value is too long"}`,
					rr.Body.String(),
				)
			},
		},
		{
			name: "wrong value in json (response_body)",
			giveBody: bytes.NewBuffer([]byte(`{
				"content_type":null,
				"status_code":null,
				"response_delay":null,
				"response_body":"` + strings.Repeat("x", 10240+1) + `"
			}`)),
			checkResult: func(t *testing.T, rr *httptest.ResponseRecorder) {
				assert.Equal(t, http.StatusBadRequest, rr.Code)
				assert.JSONEq(t,
					`{"code":400,"success":false,"message":"invalid value passed: response content is too long"}`,
					rr.Body.String(),
				)
			},
		},

		{
			name: "emulate storage error",
			giveBody: bytes.NewBuffer([]byte(`{
				"content_type":null,
				"status_code":null,
				"response_delay":null,
				"response_body":null
			}`)),
			setUp: func(s *nullStorage.Storage) {
				s.Error = errors.New("foo")
			},
			checkResult: func(t *testing.T, rr *httptest.ResponseRecorder) {
				assert.Equal(t, http.StatusInternalServerError, rr.Code)
				assert.JSONEq(t,
					`{"code":500,"success":false,"message":"foo"}`,
					rr.Body.String(),
				)
			},
		},
		{
			name: "emulate storage success",
			giveBody: bytes.NewBuffer([]byte(`{
				"content_type":null,
				"status_code":null,
				"response_delay":null,
				"response_body":null
			}`)),
			setUp: func(s *nullStorage.Storage) {
				s.SessionData = &storage.SessionData{
					UUID: "aa-bb-cc-dd",
					WebHookResponse: storage.WebHookResponse{
						Content:     "aa",
						Code:        200,
						ContentType: "foo/bar",
						DelaySec:    5,
					},
					CreatedAtUnix: 123,
				}
			},
			checkResult: func(t *testing.T, rr *httptest.ResponseRecorder) {
				assert.Equal(t, http.StatusOK, rr.Code)
				assert.JSONEq(t, `{
					"uuid":"aa-bb-cc-dd",
					"response":{
						"code":200,
						"content":"foo/bar",
						"content_type":"foo/bar",
						"delay_sec":5,
						"created_at_unix":123
					}}`,
					rr.Body.String(),
				)
			},
		},
	}

	for _, tt := range cases {
		t.Run(tt.name, func(t *testing.T) {
			var (
				req, _  = http.NewRequest(http.MethodPost, "http://testing", tt.giveBody)
				rr      = httptest.NewRecorder()
				s       = &nullStorage.Storage{}
				handler = NewHandler(s)
			)

			if tt.setUp != nil {
				tt.setUp(s)
			}

			handler.ServeHTTP(rr, req)

			tt.checkResult(t, rr)
		})
	}
}
