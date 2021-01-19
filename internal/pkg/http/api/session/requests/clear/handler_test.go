package clear

import (
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/gorilla/mux"
	"github.com/stretchr/testify/assert"
	"github.com/tarampampam/webhook-tester/internal/pkg/broadcast"
	nullStorage "github.com/tarampampam/webhook-tester/internal/pkg/storage/null"
)

func TestJSONRPCHandler_ServeHTTP(t *testing.T) {
	t.Parallel()

	var cases = []struct {
		name        string
		giveReqVars map[string]string
		setUp       func(s *nullStorage.Storage, b *broadcast.None)
		checkResult func(t *testing.T, rr *httptest.ResponseRecorder, b *broadcast.None)
	}{
		{
			name:        "without registered session UUID",
			giveReqVars: nil,
			checkResult: func(t *testing.T, rr *httptest.ResponseRecorder, _ *broadcast.None) {
				assert.Equal(t, http.StatusInternalServerError, rr.Code)
				assert.JSONEq(t,
					`{"code":500,"success":false,"message":"cannot extract session UUID"}`, rr.Body.String(),
				)
			},
		},
		{
			name:        "emulate storage error",
			giveReqVars: map[string]string{"sessionUUID": "aa-bb-cc-dd"},
			setUp: func(s *nullStorage.Storage, b *broadcast.None) {
				s.Error = errors.New("foo")
			},
			checkResult: func(t *testing.T, rr *httptest.ResponseRecorder, _ *broadcast.None) {
				assert.Equal(t, http.StatusInternalServerError, rr.Code)
				assert.JSONEq(t,
					`{"code":500,"success":false,"message":"foo"}`, rr.Body.String(),
				)
			},
		},
		{
			name:        "emulate 'not found'",
			giveReqVars: map[string]string{"sessionUUID": "aa-bb-cc-dd"},
			setUp: func(s *nullStorage.Storage, b *broadcast.None) {
				s.Error = nil
				s.Boolean = false
			},
			checkResult: func(t *testing.T, rr *httptest.ResponseRecorder, _ *broadcast.None) {
				assert.Equal(t, http.StatusNotFound, rr.Code)
				assert.JSONEq(t,
					`{"code":404,"success":false,"message":"requests for session with UUID aa-bb-cc-dd was not found"}`,
					rr.Body.String(),
				)
			},
		},
		{
			name:        "success",
			giveReqVars: map[string]string{"sessionUUID": "aa-bb-cc-dd"},
			setUp: func(s *nullStorage.Storage, b *broadcast.None) {
				s.Error = nil
				s.Boolean = true
			},
			checkResult: func(t *testing.T, rr *httptest.ResponseRecorder, b *broadcast.None) {
				time.Sleep(time.Millisecond) // goroutine must be done

				assert.Equal(t, http.StatusOK, rr.Code)
				assert.JSONEq(t, `{"success":true}`, rr.Body.String())

				ch, e := b.LastPublishedEvent()

				assert.Equal(t, "aa-bb-cc-dd", ch)
				assert.Equal(t, broadcast.NewAllRequestsDeletedEvent(), e)
			},
		},
	}

	for _, tt := range cases {
		t.Run(tt.name, func(t *testing.T) {
			var (
				req, _  = http.NewRequest(http.MethodPost, "http://testing", nil)
				rr      = httptest.NewRecorder()
				s       = &nullStorage.Storage{}
				b       = broadcast.None{}
				handler = NewHandler(s, &b)
			)

			if tt.giveReqVars != nil {
				req = mux.SetURLVars(req, tt.giveReqVars)
			}

			if tt.setUp != nil {
				tt.setUp(s, &b)
			}

			handler.ServeHTTP(rr, req)

			tt.checkResult(t, rr, &b)
		})
	}
}
