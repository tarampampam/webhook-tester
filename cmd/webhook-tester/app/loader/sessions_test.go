package loader_test

import (
	"os"
	"path/filepath"
	"testing"
	"time"

	"gh.tarampamp.am/webhook-tester/v3/cmd/webhook-tester/app/loader"
	"gh.tarampamp.am/webhook-tester/v3/internal/storage"
	"gh.tarampamp.am/webhook-tester/v3/internal/testutil/assert"
)

func writeJSON(t *testing.T, content string) string {
	t.Helper()

	f := filepath.Join(t.TempDir(), "sessions.json")

	assert.NoError(t, os.WriteFile(f, []byte(content), 0o600))

	return f
}

func TestLoadSessions(t *testing.T) {
	t.Parallel()

	for name, tt := range map[string]struct {
		giveJSON string
		want     map[string]storage.SessionResponse
		wantErr  string
	}{
		"minimal session - id only": {
			giveJSON: `{"sessions":[{"id":"aaaaaaaa-0000-0000-0000-000000000000"}]}`,
			want: map[string]storage.SessionResponse{
				"aaaaaaaa-0000-0000-0000-000000000000": {},
			},
		},
		"full session": {
			giveJSON: `{"sessions":[{
				"id":"bbbbbbbb-0000-0000-0000-000000000000",
				"code":503,
				"headers":{"X-Foo":"bar"},
				"body":"hello",
				"delay":1.5
			}]}`,
			want: map[string]storage.SessionResponse{
				"bbbbbbbb-0000-0000-0000-000000000000": {
					Code:    503,
					Headers: []storage.ResponseHeader{{Name: "X-Foo", Value: "bar"}},
					Body:    []byte("hello"),
					Delay:   1500 * time.Millisecond,
				},
			},
		},
		"multiple sessions": {
			giveJSON: `{"sessions":[
				{"id":"cccccccc-0000-0000-0000-000000000000","code":200},
				{"id":"dddddddd-0000-0000-0000-000000000000","code":404}
			]}`,
			want: map[string]storage.SessionResponse{
				"cccccccc-0000-0000-0000-000000000000": {Code: 200},
				"dddddddd-0000-0000-0000-000000000000": {Code: 404},
			},
		},
		"unknown keys are allowed ($schema)": {
			giveJSON: `{
				"$schema":"https://example.com/schema.json",
				"sessions":[{"id":"eeeeeeee-0000-0000-0000-000000000000",
				"unknown_key":42
			}]}`,
			want: map[string]storage.SessionResponse{
				"eeeeeeee-0000-0000-0000-000000000000": {},
			},
		},
		"empty sessions list": {
			giveJSON: `{"sessions":[]}`,
			want:     map[string]storage.SessionResponse{},
		},
		"body as byte array": {
			giveJSON: `{"sessions":[{"id":"ffffffff-0000-0000-0000-000000000000","body":[72,   101,108, 108,111]}]}`,
			want: map[string]storage.SessionResponse{
				"ffffffff-0000-0000-0000-000000000000": {Body: []byte("Hello")},
			},
		},
		"body as valid base64 string": {
			// "binary\x00data" base64-encoded = "YmluYXJ5AGRhdGE="
			giveJSON: `{"sessions":[{"id":"11111111-0000-0000-0000-000000000000","body":"YmluYXJ5AGRhdGE="}]}`,
			want: map[string]storage.SessionResponse{
				"11111111-0000-0000-0000-000000000000": {Body: []byte("binary\x00data")},
			},
		},
		"body as plain string (not base64)": {
			giveJSON: `{"sessions":[{"id":"22222222-0000-0000-0000-000000000000","body":"not-base64!!!"}]}`,
			want: map[string]storage.SessionResponse{
				"22222222-0000-0000-0000-000000000000": {Body: []byte("not-base64!!!")},
			},
		},
		"body null": {
			giveJSON: `{"sessions":[{"id":"33333333-0000-0000-0000-000000000000","body":null}]}`,
			want: map[string]storage.SessionResponse{
				"33333333-0000-0000-0000-000000000000": {Body: nil},
			},
		},
		"delay as float seconds": {
			giveJSON: `{"sessions":[{"id":"44444444-0000-0000-0000-000000000000","delay":0.1}]}`,
			want: map[string]storage.SessionResponse{
				"44444444-0000-0000-0000-000000000000": {Delay: 100 * time.Millisecond},
			},
		},
		"delay as duration string": {
			giveJSON: `{"sessions":[{"id":"55555555-0000-0000-0000-000000000000","delay":"2s"}]}`,
			want: map[string]storage.SessionResponse{
				"55555555-0000-0000-0000-000000000000": {Delay: 2 * time.Second},
			},
		},

		"error: missing id": {
			giveJSON: `{"sessions":[{"code":200}]}`,
			wantErr:  `session at index 0: missing required field "id"`,
		},
		"error: body is an object": {
			giveJSON: `{"sessions":[{"id":"66666666-0000-0000-0000-000000000000","body":{"key":"val"}}]}`,
			wantErr:  `session "66666666-0000-0000-0000-000000000000" body: unsupported type`,
		},
		"error: body is a bool": {
			giveJSON: `{"sessions":[{"id":"77777777-0000-0000-0000-000000000000","body":true}]}`,
			wantErr:  `session "77777777-0000-0000-0000-000000000000" body: unsupported type`,
		},
		"error: body array with out-of-range value": {
			giveJSON: `{"sessions":[{"id":"88888888-0000-0000-0000-000000000000","body":[0,256]}]}`,
			wantErr:  `session "88888888-0000-0000-0000-000000000000" body: element 1 value 256 is out of byte range`,
		},
		"error: body array with non-integer value": {
			giveJSON: `{"sessions":[{"id":"99999999-0000-0000-0000-000000000000","body":[1,2.5]}]}`,
			wantErr:  `session "99999999-0000-0000-0000-000000000000" body: element 1 value 2.5 is out of byte range`,
		},
		"error: body array with negative value": {
			giveJSON: `{"sessions":[{"id":"aaaaaaaa-1111-0000-0000-000000000000","body":[-1]}]}`,
			wantErr:  `session "aaaaaaaa-1111-0000-0000-000000000000" body: element 0 value -1 is out of byte range`,
		},
		"error: invalid delay string": {
			giveJSON: `{"sessions":[{"id":"bbbbbbbb-1111-0000-0000-000000000000","delay":"notaduration"}]}`,
			wantErr:  `invalid duration "notaduration"`,
		},
		"error: invalid json": {
			giveJSON: `{not json`,
			wantErr:  "parse sessions file",
		},
	} {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			got, err := loader.LoadSessions(writeJSON(t, tt.giveJSON))

			if tt.wantErr != "" {
				assert.ErrorContains(t, err, tt.wantErr)

				return
			}

			assert.NoError(t, err)
			assert.Equal(t, len(tt.want), len(got))

			for id, wantResp := range tt.want {
				gotResp, exists := got[id]
				assert.True(t, exists)
				assert.Equal(t, wantResp.Code, gotResp.Code)
				assert.Equal(t, wantResp.Delay, gotResp.Delay)
				assert.DeepEqual(t, wantResp.Body, gotResp.Body)
				assert.Equal(t, len(wantResp.Headers), len(gotResp.Headers))

				for _, wh := range wantResp.Headers {
					var found bool

					for _, gh := range gotResp.Headers {
						if gh.Name == wh.Name && gh.Value == wh.Value {
							found = true

							break
						}
					}

					assert.True(t, found)
				}
			}
		})
	}
}

func TestLoadSessions_FileNotFound(t *testing.T) {
	t.Parallel()

	_, err := loader.LoadSessions("/nonexistent/path/sessions.json")
	assert.ErrorContains(t, err, "read sessions file")
}
