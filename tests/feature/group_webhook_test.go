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

	t.Run("cors", func(t *testing.T) {
		t.Parallel()

		sID := ft.CreateSession(t, baseUrl, http.StatusOK, nil, nil)
		webhookURL := baseUrl + "/" + sID

		t.Run("browser preflight without requested headers", func(t *testing.T) {
			t.Parallel()

			resp := ft.MustRequest(t, http.MethodOptions, webhookURL, map[string]string{
				"Origin":                        "http://localhost:5173",
				"Access-Control-Request-Method": "POST",
			}, nil)

			assert.Equal(t, http.StatusNoContent, resp.StatusCode)
			assert.Equal(t, "*", resp.Header.Get("Access-Control-Allow-Origin"))
			assert.Equal(t, "POST", resp.Header.Get("Access-Control-Allow-Methods"))
			assert.Nil(t, resp.Header.Values("Access-Control-Allow-Headers"))
			assert.Equal(t, "86400", resp.Header.Get("Access-Control-Max-Age"))
		})

		t.Run("browser preflight mirrors Access-Control-Request-Headers", func(t *testing.T) {
			t.Parallel()

			resp := ft.MustRequest(t, http.MethodOptions, webhookURL, map[string]string{
				"Origin":                         "http://localhost:5173",
				"Access-Control-Request-Method":  "POST",
				"Access-Control-Request-Headers": "Content-Type, Authorization",
			}, nil)

			assert.Equal(t, http.StatusNoContent, resp.StatusCode)
			assert.Equal(t, "*", resp.Header.Get("Access-Control-Allow-Origin"))
			assert.Equal(t, "POST", resp.Header.Get("Access-Control-Allow-Methods"))
			assert.Equal(t, "Content-Type, Authorization", resp.Header.Get("Access-Control-Allow-Headers"))
			assert.Equal(t, "86400", resp.Header.Get("Access-Control-Max-Age"))
		})

		t.Run("OPTIONS without ACRM gets no preflight headers", func(t *testing.T) {
			t.Parallel()

			resp := ft.MustRequest(t, http.MethodOptions, webhookURL, map[string]string{
				"Origin": "http://localhost:5173",
			}, nil)

			assert.Equal(t, "*", resp.Header.Get("Access-Control-Allow-Origin"))
			assert.Nil(t, resp.Header.Values("Access-Control-Allow-Methods"))
			assert.Nil(t, resp.Header.Values("Access-Control-Allow-Headers"))
			assert.Nil(t, resp.Header.Values("Access-Control-Max-Age"))
		})

		t.Run("actual request has ACAO only", func(t *testing.T) {
			t.Parallel()

			resp := ft.MustRequest(t, http.MethodPost, webhookURL, map[string]string{
				"Origin":       "http://localhost:5173",
				"Content-Type": "application/json",
			}, []byte(`{"test":true}`))

			assert.Equal(t, "*", resp.Header.Get("Access-Control-Allow-Origin"))
			assert.Nil(t, resp.Header.Values("Access-Control-Allow-Methods"))
			assert.Nil(t, resp.Header.Values("Access-Control-Allow-Headers"))
			assert.Nil(t, resp.Header.Values("Access-Control-Max-Age"))
		})

		t.Run("preflight not captured", func(t *testing.T) {
			t.Parallel()

			isolatedSID := ft.CreateSession(t, baseUrl, http.StatusOK, nil, nil)

			ft.MustRequest(t, http.MethodOptions, baseUrl+"/"+isolatedSID, map[string]string{
				"Origin":                        "http://localhost:5173",
				"Access-Control-Request-Method": "POST",
			}, nil)

			r := ft.MustGet(t, baseUrl+"/api/session/"+isolatedSID+"/requests")

			var reqs []any
			assert.NoError(t, json.Unmarshal(ft.ReadBody(t, r), &reqs))
			assert.Equal(t, 0, len(reqs))
		})
	})

	t.Run("error-formats", func(t *testing.T) {
		t.Parallel()

		// a well-formed UUID that will never exist in storage - triggers a 404 error response
		const ghost = "deadbeef-dead-dead-dead-deaddeadbeef"

		t.Run("json", func(t *testing.T) {
			t.Parallel()

			for name, headers := range map[string]map[string]string{
				"accept application/json":             {"Accept": "application/json"},
				"accept text/json":                    {"Accept": "text/json"},
				"accept json before html - json wins": {"Accept": "application/json, text/html"},
				"accept with quality param":           {"Accept": "application/json; q=0.9"},
				"accept wildcard, content-type json":  {"Accept": "*/*", "Content-Type": "application/json"},
				"no accept, content-type json":        {"Content-Type": "application/json"},
			} {
				t.Run(name, func(t *testing.T) {
					t.Parallel()

					resp := ft.MustRequest(t, http.MethodGet, baseUrl+"/"+ghost, headers, nil)
					body := ft.ReadBody(t, resp)

					assert.Equal(t, http.StatusNotFound, resp.StatusCode)
					assert.True(t, strings.HasPrefix(resp.Header.Get("Content-Type"), "application/json"))
					assert.IsJSON(t, body)
					assert.True(t, len(ft.JSONPath[string](t, body, "error")) > 0)
					assert.True(t, len(ft.JSONPath[string](t, body, "message")) > 0)
					assert.True(t, len(ft.JSONPath[string](t, body, "powered_by")) > 0)
				})
			}
		})

		t.Run("html", func(t *testing.T) {
			t.Parallel()

			for name, headers := range map[string]map[string]string{
				"accept text/html":                    {"Accept": "text/html"},
				"accept html before json - html wins": {"Accept": "text/html, application/json"},
				"accept wildcard, content-type html":  {"Accept": "*/*", "Content-Type": "text/html"},
				"no accept, content-type html":        {"Content-Type": "text/html"},
			} {
				t.Run(name, func(t *testing.T) {
					t.Parallel()

					resp := ft.MustRequest(t, http.MethodGet, baseUrl+"/"+ghost, headers, nil)
					body := ft.ReadBody(t, resp)

					assert.Equal(t, http.StatusNotFound, resp.StatusCode)
					assert.True(t, strings.HasPrefix(resp.Header.Get("Content-Type"), "text/html"))
					assert.Contains(t, body, "<!doctype html>")
				})
			}
		})

		t.Run("plain", func(t *testing.T) {
			t.Parallel()

			for name, headers := range map[string]map[string]string{
				"accept text/plain":                    {"Accept": "text/plain"},
				"no accept, content-type plain":        {"Content-Type": "text/plain"},
				"no accept, unrecognized content-type": {"Content-Type": "application/xml"},
				"no accept, no content-type":           {},
			} {
				t.Run(name, func(t *testing.T) {
					t.Parallel()

					resp := ft.MustRequest(t, http.MethodGet, baseUrl+"/"+ghost, headers, nil)
					body := ft.ReadBody(t, resp)

					assert.Equal(t, http.StatusNotFound, resp.StatusCode)
					assert.True(t, strings.HasPrefix(resp.Header.Get("Content-Type"), "text/plain"))
					assert.Contains(t, body, "WebHook: ")
				})
			}
		})
	})
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
