package feature_test

import (
	"net/http"
	"testing"

	"gh.tarampamp.am/webhook-tester/v3/internal/testutil/assert"
	"gh.tarampamp.am/webhook-tester/v3/internal/testutil/random"
	ft "gh.tarampamp.am/webhook-tester/v3/tests/feature/featuretest"
)

func groupTestSPA(t *testing.T, baseUrl string) {
	t.Helper()

	for target, url := range map[string]string{
		"index":     baseUrl,
		"not found": baseUrl + "/foo" + random.String(8),
	} {
		t.Run(target, func(t *testing.T) {
			t.Parallel()

			resp := ft.MustGet(t, url)

			assert.Equal(t, http.StatusOK, resp.StatusCode)
			assert.Contains(t, resp.Header.Get("Content-Type"), "text/html")
			assert.Contains(t, string(ft.ReadBody(t, resp)), "<html")
		})
	}

	t.Run("robots.txt", func(t *testing.T) {
		t.Parallel()

		resp := ft.MustGet(t, baseUrl+"/robots.txt")

		assert.Equal(t, http.StatusOK, resp.StatusCode)
		assert.Contains(t, resp.Header.Get("Content-Type"), "text/plain")
		assert.Contains(t, string(ft.ReadBody(t, resp)), "User-agent: *")
	})
}
