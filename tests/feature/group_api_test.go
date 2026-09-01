package feature_test

import (
	"net/http"
	"testing"

	"gh.tarampamp.am/webhook-tester/v3/internal/testutil/assert"
	"gh.tarampamp.am/webhook-tester/v3/internal/testutil/random"
	ft "gh.tarampamp.am/webhook-tester/v3/tests/feature/featuretest"
)

func groupTestAPI(t *testing.T, baseUrl string) {
	t.Helper()

	t.Run("not found", func(t *testing.T) {
		t.Parallel()

		resp := ft.MustGet(t, baseUrl+"/////api/foo"+random.String(8))
		body := ft.ReadBody(t, resp)

		assert.Equal(t, http.StatusNotFound, resp.StatusCode)
		assert.Contains(t, resp.Header.Get("Content-Type"), "application/json")
		assert.IsJSON(t, body)
		assert.Equal(t, ft.JSONPath[string](t, body, "error"), "handler not found")
	})

	t.Run("ready", func(t *testing.T) {
		t.Parallel()

		resp := ft.MustGet(t, baseUrl+"/ready")

		assert.Equal(t, http.StatusOK, resp.StatusCode)
		assert.Contains(t, resp.Header.Get("Content-Type"), "text/plain")
		assert.Contains(t, ft.ReadBody(t, resp), "OK")
	})

	t.Run("settings", func(t *testing.T) {
		t.Parallel()

		resp := ft.MustGet(t, baseUrl+"/api/settings")

		assert.Equal(t, http.StatusOK, resp.StatusCode)
		assert.Contains(t, resp.Header.Get("Content-Type"), "application/json")
		assert.IsJSON(t, ft.ReadBody(t, resp))
		// TODO: add more assertions about the settings response
	})
}
