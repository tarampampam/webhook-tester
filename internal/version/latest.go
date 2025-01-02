package version

import (
	"context"
	"fmt"
	"net/http"
	"net/url"
	"strings"
	"time"
)

type httpClient interface {
	Do(req *http.Request) (*http.Response, error)
}

// Latest returns the latest release tag of the "tarampampam/webhook-tester" repository.
//
// Optionally, you can pass a custom HTTP client to use for the request. If not provided, the default client with
// a 15-second timeout will be used.
//
// The 'v' prefix will be removed from the tag if it exists.
func Latest(ctx context.Context, useClient ...httpClient) (string, error) {
	var doer httpClient

	if len(useClient) > 0 {
		doer = useClient[0]
	} else {
		doer = &http.Client{
			Timeout:   time.Second * 15, //nolint:mnd
			Transport: &http.Transport{Proxy: http.ProxyFromEnvironment},
			CheckRedirect: func(*http.Request, []*http.Request) error {
				return http.ErrUseLastResponse // disable redirects
			},
		}
	}

	const ownerAndRepo = "tarampampam/webhook-tester"

	// use the "magic" GitHub link to get the latest release tag (it returns a 302 redirect with the tag in
	// the location header). this "hack" allows us to avoid the GitHub API rate limits
	req, reqErr := http.NewRequestWithContext(ctx,
		http.MethodGet,
		fmt.Sprintf("https://github.com/%s/releases/latest", ownerAndRepo),
		http.NoBody,
	)
	if reqErr != nil {
		return "", reqErr
	}

	// send the request
	resp, respErr := doer.Do(req)
	if respErr != nil {
		return "", respErr
	}

	// body is not interesting for us
	if resp.Body != nil {
		_ = resp.Body.Close()
	}

	// check the status code
	if resp.StatusCode != http.StatusFound {
		return "", fmt.Errorf("unexpected status code: %d", resp.StatusCode)
	}

	// parse the location header
	u, uErr := url.Parse(resp.Header.Get("Location"))
	if uErr != nil {
		return "", fmt.Errorf("parsing location header failed: %w", uErr)
	}

	// split path by slashes: [owner repo releases tag v1.2.3]
	var parts = strings.Split(strings.TrimLeft(u.Path, "/"), "/")
	if len(parts) < 5 { //nolint:mnd
		return "", fmt.Errorf("unexpected location path: %s", u.Path)
	}

	// pick the 4th segment (tag)
	var tag = parts[4]

	// if the tag starts with "v" - remove it
	if len(tag) > 1 && ((tag[0] == 'v' || tag[0] == 'V') && (tag[1] >= '0' && tag[1] <= '9')) {
		return tag[1:], nil
	}

	return tag, nil
}
