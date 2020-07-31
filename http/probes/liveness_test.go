package probes

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestLiveness_ServeHTTP(t *testing.T) {
	var (
		req, _  = http.NewRequest(http.MethodPost, "http://testing", nil)
		rr      = httptest.NewRecorder()
		handler = NewLivenessHandler()
	)

	handler.ServeHTTP(rr, req)

	assert.Equal(t, http.StatusOK, rr.Code)
}
