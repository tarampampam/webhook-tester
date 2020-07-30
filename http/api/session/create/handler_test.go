package create

import (
	"io"
	"net/http"
	"net/http/httptest"
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
		setUp       func(s storage.Storage)
		checkResult func(t *testing.T, rr *httptest.ResponseRecorder)
	}{
		{
			name:     "nil body",
			giveBody: nil,
			checkResult: func(t *testing.T, rr *httptest.ResponseRecorder) {
				assert.JSONEq(t,
					`{"code":400,"success":false,"message":"empty request body"}`,
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
