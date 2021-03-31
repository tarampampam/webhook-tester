package responder_test

import (
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/tarampampam/webhook-tester/internal/pkg/http/responder"

	"github.com/stretchr/testify/assert"
)

type jsonable struct {
	Foo string `json:"foo"`
}

func (e jsonable) StatusCode() int { return 123 }

func (e jsonable) ToJSON() ([]byte, error) { return json.Marshal(e) }

type jsonableNoCode struct {
	Foo string `json:"foo"`
}

func (e jsonableNoCode) ToJSON() ([]byte, error) { return json.Marshal(e) }

type jsonableEncodingFailed struct {
	Foo string `json:"foo"`
}

func (e jsonableEncodingFailed) ToJSON() ([]byte, error) { return nil, errors.New("fake error") }

func TestJSONSuccess(t *testing.T) {
	cases := []struct {
		name        string
		giveModel   interface{ ToJSON() ([]byte, error) }
		wantCode    int
		wantContent string
	}{
		{
			name:        "jsonable struct",
			giveModel:   jsonable{"bar"},
			wantCode:    123,
			wantContent: `{"foo":"bar"}`,
		},
		{
			name:        "jsonableNoCode struct",
			giveModel:   jsonableNoCode{"bar"},
			wantCode:    http.StatusOK,
			wantContent: `{"foo":"bar"}`,
		},
		{
			name:        "jsonableEncodingFailed struct",
			giveModel:   jsonableEncodingFailed{"bar"},
			wantCode:    http.StatusInternalServerError,
			wantContent: `{"success":false,"message":"fake error"}`,
		},
	}

	for _, tt := range cases {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			rr := httptest.NewRecorder()

			responder.JSON(rr, tt.giveModel)

			assert.Equal(t, "application/json; charset=utf-8", rr.Header().Get("Content-Type"))
			assert.Equal(t, tt.wantCode, rr.Code)
			assert.Equal(t, tt.wantContent, rr.Body.String())
		})
	}
}
