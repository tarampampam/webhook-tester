package probes

import (
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
	nullStorage "github.com/tarampampam/webhook-tester/internal/pkg/storage/null"
)

func TestReadiness_ServeHTTP(t *testing.T) {
	t.Parallel()

	var cases = []struct {
		name        string
		setUp       func(s *nullStorage.Storage)
		checkResult func(t *testing.T, rr *httptest.ResponseRecorder)
	}{
		{
			name: "ok",
			setUp: func(s *nullStorage.Storage) {
				s.Error = nil
			},
			checkResult: func(t *testing.T, rr *httptest.ResponseRecorder) {
				assert.Equal(t, http.StatusOK, rr.Code)
				assert.Equal(t, "", rr.Body.String())
			},
		},
		{
			name: "storage error",
			setUp: func(s *nullStorage.Storage) {
				s.Error = errors.New("foo")
			},
			checkResult: func(t *testing.T, rr *httptest.ResponseRecorder) {
				assert.Equal(t, http.StatusServiceUnavailable, rr.Code)
				assert.Equal(t, "foo\n", rr.Body.String())
			},
		},
	}

	for _, tt := range cases {
		t.Run(tt.name, func(t *testing.T) {
			var (
				req, _  = http.NewRequest(http.MethodPost, "http://testing", nil)
				rr      = httptest.NewRecorder()
				s       = &nullStorage.Storage{}
				handler = NewReadinessHandler(s)
			)

			if tt.setUp != nil {
				tt.setUp(s)
			}

			handler.ServeHTTP(rr, req)

			tt.checkResult(t, rr)
		})
	}
}
