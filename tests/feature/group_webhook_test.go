package feature_test

import (
	"encoding/base64"
	"encoding/json"
	"net/http"
	"strconv"
	"strings"
	"testing"

	"gh.tarampamp.am/webhook-tester/v3/internal/testutil/assert"
	"gh.tarampamp.am/webhook-tester/v3/internal/testutil/random"
	ft "gh.tarampamp.am/webhook-tester/v3/tests/feature/featuretest"
)

func groupTestWebhook(t *testing.T, baseUrl string) {
	t.Helper()

	// curated set covering: normal 2xx, no-body 2xx, 4xx, 5xx, and boundary values
	for _, statusCode := range []int{200, 201, 204, 400, 418, 500, 530} {
		t.Run(strconv.Itoa(statusCode), func(t *testing.T) {
			t.Parallel()

			var (
				respHeaderName  = "X-Session-" + random.String(6)
				respHeaderValue = random.String(10)
				respBody        = []byte(`{"c":` + strconv.Itoa(statusCode) + `}`)
			)

			sID := ft.CreateSession(t, baseUrl, statusCode,
				map[string]string{respHeaderName: respHeaderValue},
				respBody,
			)

			// initially no requests
			{
				r := ft.MustGet(t, baseUrl+"/api/session/"+sID+"/requests")
				body := ft.ReadBody(t, r)

				assert.Equal(t, http.StatusOK, r.StatusCode)
				assert.IsJSON(t, body)

				var reqs []any
				assert.NoError(t, json.Unmarshal(body, &reqs))
				assert.Equal(t, 0, len(reqs))
			}

			// for these status codes the HTTP spec forbids a response body; the Go HTTP client returns none
			bodyMustBeEmpty := statusCode == 204 || statusCode == 304

			for _, method := range []string{
				http.MethodGet, http.MethodPost, http.MethodPut,
				http.MethodDelete, http.MethodPatch, http.MethodOptions,
			} {
				for _, path := range []string{"", "/foo", "////bar////////baz?yes=no"} {
					for _, payload := range []string{"", strings.Repeat("foobar", 100)} {
						t.Run(method+"|"+path+"|"+strconv.Itoa(len(payload)), func(t *testing.T) {
							reqHeaderValue := random.String(10)

							resp := ft.MustRequest(t, method,
								baseUrl+"/"+sID+path,
								map[string]string{"X-Test-Key": reqHeaderValue},
								[]byte(payload),
							)
							body := ft.ReadBody(t, resp)

							// response from the server must match the session's configured values
							assert.Equal(t, statusCode, resp.StatusCode)

							if bodyMustBeEmpty {
								assert.Equal(t, 0, len(body))
							} else {
								assert.DeepEqual(t, respBody, body)
							}

							assert.Equal(t, "*", resp.Header.Get("Access-Control-Allow-Origin"))
							assert.Equal(t, "*", resp.Header.Get("Access-Control-Allow-Methods"))
							assert.Equal(t, "*", resp.Header.Get("Access-Control-Allow-Headers"))
							assert.Equal(t, respHeaderValue, resp.Header.Get(respHeaderName))

							rID := resp.Header.Get("X-Wh-Request-Id")
							assert.Equal(t, 36, len(rID))

							// verify the captured request data
							captResp := ft.MustGet(t, baseUrl+"/api/session/"+sID+"/requests/"+rID)
							captBody := ft.ReadBody(t, captResp)

							assert.Equal(t, http.StatusOK, captResp.StatusCode)
							assert.IsJSON(t, captBody)
							assert.Equal(t, 36, len(ft.JSONPath[string](t, captBody, "uuid")))
							assert.True(t, len(ft.JSONPath[string](t, captBody, "client_address")) > 0)
							assert.Equal(t, method, ft.JSONPath[string](t, captBody, "method"))
							assert.True(t, strings.HasSuffix(ft.JSONPath[string](t, captBody, "url"), "/"+sID+path))

							decodedPayload, decErr := base64.StdEncoding.DecodeString(
								ft.JSONPath[string](t, captBody, "request_payload_base64"),
							)
							assert.NoError(t, decErr)
							assert.Equal(t, payload, string(decodedPayload))

							rawHeaders := ft.JSONPath[[]any](t, captBody, "headers")
							assert.True(t, capturedHeadersContain(rawHeaders, "X-Test-Key", reqHeaderValue))
						})
					}
				}
			}
		})
	}
}

// capturedHeadersContain checks if the captured headers list contains an entry with the given name and value.
func capturedHeadersContain(headers []any, name, value string) bool {
	for _, h := range headers {
		m, ok := h.(map[string]any)
		if !ok {
			continue
		}

		if m["name"] == name && m["value"] == value {
			return true
		}
	}

	return false
}
