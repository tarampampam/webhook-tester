package ft

import (
	"encoding/base64"
	"net/http"
	"slices"
	"strings"
	"testing"
	"time"

	"gh.tarampamp.am/webhook-tester/v3/internal/testutil/assert"
)

// CreateSession creates a session with the given parameters and asserts that the response is correct.
func CreateSession(t *testing.T, baseUrl string, status int, headers map[string]string, body []byte) string {
	t.Helper()

	type hdr struct {
		Name  string `json:"name"`
		Value string `json:"value"`
	}

	hdrs := make([]hdr, 0, len(headers))
	for k, v := range headers {
		hdrs = append(hdrs, hdr{Name: k, Value: v})
	}

	slices.SortFunc(hdrs, func(a, b hdr) int { return strings.Compare(a.Name, b.Name) })

	var (
		b64  = base64.StdEncoding.EncodeToString(body)
		resp = MustPost(t,
			baseUrl+"/api/session",
			map[string]string{"Content-Type": "application/json"},
			ToJSON(t, map[string]any{
				"status_code":          status,
				"headers":              hdrs,
				"delay":                0,
				"response_body_base64": b64,
			}),
		)
		respBody = ReadBody(t, resp)
	)

	assert.Equal(t, http.StatusCreated, resp.StatusCode)
	assert.IsJSON(t, respBody)
	assert.InDelta(t,
		float64(time.Now().UnixMilli()),
		JSONPath[float64](t, respBody, "created_at_unix_milli"),
		1000, //nolint:mnd // 1 second in milliseconds
	)

	return JSONPath[string](t, respBody, "uuid")
}
