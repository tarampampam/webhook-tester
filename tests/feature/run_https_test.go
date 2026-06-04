package feature_test

import (
	"net/http"
	"strings"
	"testing"

	"gh.tarampamp.am/webhook-tester/v3/internal/testutil/assert"
	ft "gh.tarampamp.am/webhook-tester/v3/tests/feature/featuretest"
)

// TestHTTPS verifies TLS support - self-signed certificate, HTTP-to-HTTPS redirect on the same port,
// and a minimal end-to-end webhook round-trip over HTTPS.
func TestHTTPS(t *testing.T) {
	t.Parallel()

	baseUrl := ft.MustRunApp(t, "--self-signed-tls")

	assert.True(t, strings.HasPrefix(baseUrl, "https://")) // just in case

	baseUrlHttp := strings.Replace(baseUrl, "https://", "http://", 1)

	t.Run("tls connectivity", func(t *testing.T) {
		t.Parallel()

		resp := ft.MustGet(t, baseUrl+"/ready")

		assert.Equal(t, http.StatusOK, resp.StatusCode)
		assert.Contains(t, ft.ReadBody(t, resp), "OK")
	})

	t.Run("http to https redirect", func(t *testing.T) {
		t.Parallel()

		giveReqUrl := baseUrlHttp + "/ready"  // HTTP
		wantRedirectUrl := baseUrl + "/ready" // HTTPS

		resp := ft.MustGet(t, giveReqUrl)

		assert.Equal(t, http.StatusTemporaryRedirect, resp.StatusCode)
		assert.Equal(t, wantRedirectUrl, resp.Header.Get("Location"))
	})

	t.Run("webhook round trip", func(t *testing.T) {
		t.Parallel()

		sID := ft.CreateSession(t, baseUrl, http.StatusOK, nil, []byte("tls-ok"))

		resp := ft.MustRequest(t, http.MethodPost, baseUrl+"/"+sID, nil, []byte("ping"))
		body := ft.ReadBody(t, resp)

		assert.Equal(t, http.StatusOK, resp.StatusCode)
		assert.DeepEqual(t, []byte("tls-ok"), body)
	})
}
