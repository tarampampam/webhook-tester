package healthz

import (
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
)

type fakeChecker struct{ err error }

func (c *fakeChecker) Check() error { return c.err }

func TestNewHandlerNoError(t *testing.T) {
	var (
		req, _ = http.NewRequest(http.MethodGet, "http://testing", http.NoBody)
		rr     = httptest.NewRecorder()
	)

	NewHandler(&fakeChecker{err: nil})(rr, req)

	assert.Equal(t, rr.Code, http.StatusOK)
	assert.Empty(t, rr.Body.Bytes())
}

func TestNewHandlerError(t *testing.T) {
	var (
		req, _ = http.NewRequest(http.MethodGet, "http://testing?foo=bar", http.NoBody)
		rr     = httptest.NewRecorder()
	)

	NewHandler(&fakeChecker{err: errors.New("foo")})(rr, req)

	assert.Equal(t, rr.Code, http.StatusServiceUnavailable)
	assert.Equal(t, "foo", rr.Body.String())
}
