package version_test

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/tarampampam/webhook-tester/internal/pkg/http/handlers/api/version"
)

func TestNewHandler(t *testing.T) {
	var (
		req, _ = http.NewRequest(http.MethodGet, "http://testing", nil)
		rr     = httptest.NewRecorder()
	)

	version.NewHandler("1.2.3@foo")(rr, req)

	assert.Equal(t, rr.Code, http.StatusOK)
	assert.JSONEq(t, `{"version":"1.2.3@foo"}`, rr.Body.String())
}
