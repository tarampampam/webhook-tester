package feature_test

import (
	"strings"
	"testing"

	"gh.tarampamp.am/webhook-tester/v3/internal/testutil/assert"
	ft "gh.tarampamp.am/webhook-tester/v3/tests/feature/featuretest"
)

func groupTestAPISession(t *testing.T, baseUrl string) {
	t.Helper()

	t.Run("create", func(t *testing.T) {
		t.Parallel()

		t.Run("negative", func(t *testing.T) {
			t.Parallel()

			for name, tc := range map[string]struct {
				payload       map[string]any
				wantErrSubstr string
			}{
				"too small status code": {
					payload:       map[string]any{"status_code": 99, "headers": []any{}, "delay": 0, "response_body_base64": ""},
					wantErrSubstr: "wrong status code",
				},
				"too big status code": {
					payload:       map[string]any{"status_code": 531, "headers": []any{}, "delay": 0, "response_body_base64": ""},
					wantErrSubstr: "wrong status code",
				},
				"invalid header name": {
					payload: map[string]any{
						"status_code":          200,
						"headers":              []any{map[string]any{"name": "", "value": "bar"}},
						"delay":                0,
						"response_body_base64": "",
					},
					wantErrSubstr: "header key length",
				},
				"invalid header value": {
					payload: map[string]any{
						"status_code":          200,
						"headers":              []any{map[string]any{"name": "foo", "value": strings.Repeat("x", 2049)}},
						"delay":                0,
						"response_body_base64": "",
					},
					wantErrSubstr: "header value length",
				},
				"negative delay": {
					payload:       map[string]any{"status_code": 200, "headers": []any{}, "delay": -1, "response_body_base64": ""},
					wantErrSubstr: "delay",
				},
				"too big delay": {
					payload:       map[string]any{"status_code": 200, "headers": []any{}, "delay": 31, "response_body_base64": ""},
					wantErrSubstr: "delay",
				},
				"invalid base64": {
					payload: map[string]any{
						"status_code":          200,
						"headers":              []any{},
						"delay":                0,
						"response_body_base64": "not-valid-base64!!!",
					},
					wantErrSubstr: "cannot decode response body",
				},
			} {
				t.Run(name, func(t *testing.T) {
					t.Parallel()

					resp := ft.MustPost(t, baseUrl+"/api/session",
						map[string]string{"Content-Type": "application/json"},
						ft.ToJSON(t, tc.payload),
					)
					body := ft.ReadBody(t, resp)

					assert.Equal(t, 400, resp.StatusCode)
					assert.Contains(t, resp.Header.Get("Content-Type"), "application/json")
					assert.IsJSON(t, body)
					assert.Contains(t, string(body), tc.wantErrSubstr)
				})
			}
		})
	})

	t.Run("get", func(t *testing.T) {
		t.Parallel()

		t.Run("not found", func(t *testing.T) {
			t.Parallel()

			resp := ft.MustGet(t, baseUrl+"/api/session/00000000-0000-0000-0000-000000000000")
			body := ft.ReadBody(t, resp)

			assert.Equal(t, 404, resp.StatusCode)
			assert.Contains(t, resp.Header.Get("Content-Type"), "application/json")
			assert.IsJSON(t, body)
			assert.Contains(t, string(body), "session not found")
		})

		t.Run("invalid id format", func(t *testing.T) {
			t.Parallel()

			resp := ft.MustGet(t, baseUrl+"/api/session/foobar")
			body := ft.ReadBody(t, resp)

			assert.Equal(t, 400, resp.StatusCode)
			assert.Contains(t, resp.Header.Get("Content-Type"), "application/json")
			assert.IsJSON(t, body)
			assert.Contains(t, string(body), "invalid session ID")
		})
	})

	t.Run("delete", func(t *testing.T) {
		t.Parallel()

		sID := ft.CreateSession(t, baseUrl, 200, nil, nil)

		// verify the session exists
		assert.Equal(t, 200, ft.MustGet(t, baseUrl+"/api/session/"+sID).StatusCode)

		// delete the session
		delResp := ft.MustDelete(t, baseUrl+"/api/session/"+sID)
		delBody := ft.ReadBody(t, delResp)

		assert.Equal(t, 200, delResp.StatusCode)
		assert.Contains(t, delResp.Header.Get("Content-Type"), "application/json")
		assert.IsJSON(t, delBody)
		assert.Equal(t, true, ft.JSONPath[bool](t, delBody, "success"))

		// session should be gone now
		assert.Equal(t, 404, ft.MustGet(t, baseUrl+"/api/session/"+sID).StatusCode)

		// deleting again should also be 404
		assert.Equal(t, 404, ft.MustDelete(t, baseUrl+"/api/session/"+sID).StatusCode)
	})
}
