package create

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/alicebob/miniredis"
	"github.com/go-redis/redis/v8"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/tarampampam/webhook-tester/internal/pkg/storage"
)

func TestHandler_ServeHTTPSessionCreation(t *testing.T) {
	mini, err := miniredis.Run()
	assert.NoError(t, err)

	defer mini.Close()

	s := storage.NewRedisStorage(context.TODO(), redis.NewClient(&redis.Options{Addr: mini.Addr()}), time.Minute, 1)

	var (
		req, _ = http.NewRequest(http.MethodPost, "http://test", bytes.NewBuffer([]byte(`{
			"content_type":null,
			"status_code":null,
			"response_delay":null,
			"response_body":null
		}`)))
		rr = httptest.NewRecorder()
		h  = NewHandler(s)
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

	_, err = uuid.Parse(resp.UUID)
	assert.NoError(t, err)

	assert.Equal(t, "", resp.ResponseSettings.Content)
	assert.Equal(t, uint16(200), resp.ResponseSettings.Code)
	assert.Equal(t, "text/plain", resp.ResponseSettings.ContentType)
	assert.Equal(t, uint8(0), resp.ResponseSettings.DelaySec)
	assert.Equal(t, time.Now().Unix(), resp.ResponseSettings.CreatedAtUnix)
}

func TestHandler_ServeHTTPErrors(t *testing.T) {
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
			wantResponseJSON: `{"code":400,"success":false,"message":"invalid value passed: delay is too much"}`,
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
			wantResponseJSON: `{"code":400,"success":false,"message":"invalid value passed: wrong status code value"}`,
		},
		{
			name: "wrong value in json (content_type)",
			giveRequestBody: func() io.Reader {
				return bytes.NewBuffer([]byte(`{
					"content_type":"` + strings.Repeat("x", 512) + `",
					"status_code":null,
					"response_delay":null,
					"response_body":""
				}`))
			},
			wantStatusCode:   http.StatusBadRequest,
			wantResponseJSON: `{"code":400,"success":false,"message":"invalid value passed: content-type value is too long"}`,
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
			wantResponseJSON: `{"code":400,"success":false,"message":"invalid value passed: response content is too long"}`,
		},
	}

	for _, tt := range cases {
		t.Run(tt.name, func(t *testing.T) {
			mini, err := miniredis.Run()
			assert.NoError(t, err)

			defer mini.Close()

			s := storage.NewRedisStorage(context.TODO(), redis.NewClient(&redis.Options{Addr: mini.Addr()}), time.Minute, 10)

			var (
				req, _  = http.NewRequest(http.MethodPost, "http://test", tt.giveRequestBody())
				rr      = httptest.NewRecorder()
				handler = NewHandler(s)
			)

			handler.ServeHTTP(rr, req)

			assert.Equal(t, tt.wantStatusCode, rr.Code)
			assert.JSONEq(t, tt.wantResponseJSON, rr.Body.String())
		})
	}
}
