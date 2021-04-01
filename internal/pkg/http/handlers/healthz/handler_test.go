package healthz_test

import (
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/tarampampam/webhook-tester/internal/pkg/http/handlers/healthz"
)

type fakeChecker struct{ err error }

func (c *fakeChecker) Check() error { return c.err }

func TestNewHandlerNoError(t *testing.T) {
	var (
		req, _ = http.NewRequest(http.MethodGet, "http://testing", http.NoBody)
		rr     = httptest.NewRecorder()
	)

	healthz.NewHandler(&fakeChecker{err: nil})(rr, req)

	assert.Equal(t, rr.Code, http.StatusOK)
	assert.Empty(t, rr.Body.Bytes())
}

func TestNewHandlerError(t *testing.T) {
	var (
		req, _ = http.NewRequest(http.MethodGet, "http://testing?foo=bar", http.NoBody)
		rr     = httptest.NewRecorder()
	)

	healthz.NewHandler(&fakeChecker{err: errors.New("foo")})(rr, req)

	assert.Equal(t, rr.Code, http.StatusServiceUnavailable)
	assert.Equal(t, "foo", rr.Body.String())
}
