package version_test

import (
	"context"
	"net/http"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"gh.tarampamp.am/webhook-tester/v2/internal/version"
)

type httpClientFunc func(*http.Request) (*http.Response, error)

func (f httpClientFunc) Do(req *http.Request) (*http.Response, error) { return f(req) }

func TestLatest(t *testing.T) {
	t.Parallel()

	for name, tt := range map[string]struct {
		giveStatusCode int
		giveLocation   string

		wantVersion     string
		wantErrorSubstr string
	}{
		"success": {
			giveStatusCode: http.StatusFound,
			giveLocation:   "https://github.com/tarampampam/webhook-tester/releases/tag/V1.2.0/foo/bar?baz=qux#quux",
			wantVersion:    "1.2.0",
		},
		"success without v prefix": {
			giveStatusCode: http.StatusFound,
			giveLocation:   "https://github.com/tarampampam/webhook-tester/releases/tag/1.2.0/foo/bar?baz=qux#quux",
			wantVersion:    "1.2.0",
		},

		"unexpected status code": {
			giveStatusCode:  http.StatusNotFound,
			wantErrorSubstr: "unexpected status code: 404",
		},
		"redirect location is malformed": {
			giveStatusCode:  http.StatusFound,
			giveLocation:    "qwe",
			wantErrorSubstr: "unexpected location path: qwe",
		},
		"too short location link": {
			giveStatusCode:  http.StatusFound,
			giveLocation:    "https://github.com/owner/repo/foo",
			wantErrorSubstr: "unexpected location path: /owner/repo/foo",
		},
	} {
		t.Run(name, func(t *testing.T) {
			var client httpClientFunc = func(req *http.Request) (*http.Response, error) {
				return &http.Response{
					StatusCode: tt.giveStatusCode,
					Header:     http.Header{"Location": []string{tt.giveLocation}},
				}, nil
			}

			latest, err := version.Latest(context.Background(), client)

			if tt.wantErrorSubstr == "" {
				require.NoError(t, err)
				assert.Equal(t, tt.wantVersion, latest)
			} else {
				require.Error(t, err)
				assert.ErrorContains(t, err, tt.wantErrorSubstr)
				assert.Empty(t, latest)
			}
		})
	}
}
