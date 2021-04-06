package create_test

import (
	"bytes"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/tarampampam/webhook-tester/internal/pkg/http/handlers/api/session/create"
	"github.com/tarampampam/webhook-tester/internal/pkg/storage"
)

func TestHandlerErrors(t *testing.T) {
	s := storage.NewInMemory(time.Minute, 1)
	defer s.Close()

	h := create.NewHandler(s)

	var cases = []struct {
		name             string
		giveRequestBody  func() io.Reader
		wantStatusCode   int
		wantResponseJSON string
	}{
		{
			name:             "nil body",
			giveRequestBody:  func() io.Reader { return nil },
			wantStatusCode:   http.StatusBadRequest,
			wantResponseJSON: `{"code":400,"success":false,"message":"empty request body"}`,
		},
		{
			name: "wrong json",
			giveRequestBody: func() io.Reader {
				return bytes.NewBuffer([]byte(`{json`))
			},
			wantStatusCode:   http.StatusBadRequest,
			wantResponseJSON: `{"code":400,"success":false,"message":"cannot parse passed json"}`,
		},
		{
			name: "wrong value in correct json struct (unmarshal error)",
			giveRequestBody: func() io.Reader {
				return bytes.NewBuffer([]byte(`{
					"content_type":null,
					"status_code":null,
					"response_delay":-9999,
					"response_body":""
				}`))
			},
			wantStatusCode:   http.StatusBadRequest,
			wantResponseJSON: `{"code":400,"success":false,"message":"cannot parse passed json"}`,
		},
		{
			name: "wrong value in json (response_delay)",
			giveRequestBody: func() io.Reader {
				return bytes.NewBuffer([]byte(`{
					"content_type":null,
					"status_code":null,
					"response_delay":99,
					"response_body":""
				}`))
			},
			wantStatusCode:   http.StatusBadRequest,
			wantResponseJSON: `{"code":400,"success":false,"message":"wrong request: delay is too much"}`,
		},
		{
			name: "wrong value in json (status_code)",
			giveRequestBody: func() io.Reader {
				return bytes.NewBuffer([]byte(`{
					"content_type":null,
					"status_code":1,
					"response_delay":null,
					"response_body":""
				}`))
			},
			wantStatusCode:   http.StatusBadRequest,
			wantResponseJSON: `{"code":400,"success":false,"message":"wrong request: wrong status code"}`,
		},
		{
			name: "wrong value in json (content_type)",
			giveRequestBody: func() io.Reader {
				return bytes.NewBuffer([]byte(`{
					"content_type":"` + strings.Repeat("x", 32+1) + `",
					"status_code":null,
					"response_delay":null,
					"response_body":""
				}`))
			},
			wantStatusCode:   http.StatusBadRequest,
			wantResponseJSON: `{"code":400,"success":false,"message":"wrong request: content-type value is too large"}`,
		},
		{
			name: "wrong value in json (response_body)",
			giveRequestBody: func() io.Reader {
				return bytes.NewBuffer([]byte(`{
					"content_type":null,
					"status_code":null,
					"response_delay":null,
					"response_body":"` + strings.Repeat("x", 10240+1) + `"
				}`))
			},
			wantStatusCode:   http.StatusBadRequest,
			wantResponseJSON: `{"code":400,"success":false,"message":"wrong request: response content is too large"}`,
		},
	}

	for _, tt := range cases {
		t.Run(tt.name, func(t *testing.T) {
			var (
				req, _ = http.NewRequest(http.MethodPost, "http://test", tt.giveRequestBody())
				rr     = httptest.NewRecorder()
			)

			h.ServeHTTP(rr, req)

			assert.Equal(t, tt.wantStatusCode, rr.Code)
			assert.JSONEq(t, tt.wantResponseJSON, rr.Body.String())
		})
	}
}

func TestHandlerSessionCreation(t *testing.T) {
	s := storage.NewInMemory(time.Minute, 1)
	defer s.Close()

	var (
		req, _ = http.NewRequest(http.MethodPost, "http://test", bytes.NewBuffer([]byte(`{
			"content_type":null,
			"status_code":null,
			"response_delay":null,
			"response_body":null
		}`)))
		rr = httptest.NewRecorder()
		h  = create.NewHandler(s)
	)

	h.ServeHTTP(rr, req)

	assert.Equal(t, http.StatusOK, rr.Code)

	resp := struct {
		UUID             string `json:"uuid"`
		ResponseSettings struct {
			Content       string `json:"content"`
			Code          uint16 `json:"code"`
			ContentType   string `json:"content_type"`
			DelaySec      uint8  `json:"delay_sec"`
			CreatedAtUnix int64  `json:"created_at_unix"`
		} `json:"response"`
	}{}

	assert.NoError(t, json.Unmarshal(rr.Body.Bytes(), &resp))

	_, err := uuid.Parse(resp.UUID)
	assert.NoError(t, err)

	assert.Equal(t, "", resp.ResponseSettings.Content)
	assert.Equal(t, uint16(200), resp.ResponseSettings.Code)
	assert.Equal(t, "text/plain", resp.ResponseSettings.ContentType)
	assert.Equal(t, uint8(0), resp.ResponseSettings.DelaySec)
	assert.Equal(t, time.Now().Unix(), resp.ResponseSettings.CreatedAtUnix)
}
