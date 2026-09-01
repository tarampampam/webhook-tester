package webhook

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"strconv"
	"strings"

	"gh.tarampamp.am/webhook-tester/v3/internal/appmeta"
	"gh.tarampamp.am/webhook-tester/v3/internal/logger"
)

type responseFormat byte

const (
	responseFormatPlainText responseFormat = iota
	responseFormatHTML
	responseFormatJSON
)

// renderError renders an error response in the best format, based on the request headers.
func renderError(
	log *logger.Logger,
	w http.ResponseWriter,
	r *http.Request,
	err error,
) {
	if err == nil {
		err = errors.New("unknown error")
	}

	renderErrorMessage(log, w, r, http.StatusInternalServerError, err.Error())
}

// renderErrorMessage renders an error response in the best format, based on the request headers.
func renderErrorMessage( //nolint:funlen
	log *logger.Logger,
	w http.ResponseWriter,
	r *http.Request,
	statusCode int,
	msg string,
) {
	// determine the best response format based on the request headers
	format := func() responseFormat {
		for part := range strings.SplitSeq(r.Header.Get("Accept"), ",") {
			mimeType, _, _ := strings.Cut(strings.TrimSpace(part), ";")
			if _, sub, ok := strings.Cut(mimeType, "/"); ok {
				switch name, _, _ := strings.Cut(sub, "+"); {
				case strings.EqualFold(name, "json"): // application/json, text/json
					return responseFormatJSON
				case strings.EqualFold(name, "html"): // text/html
					return responseFormatHTML
				case strings.EqualFold(name, "plain"): // text/plain
					return responseFormatPlainText
				}
			}
		}

		mimeType, _, _ := strings.Cut(r.Header.Get("Content-Type"), ";")
		if _, sub, ok := strings.Cut(mimeType, "/"); ok {
			switch name, _, _ := strings.Cut(sub, "+"); {
			case strings.EqualFold(name, "json"): // application/json, text/json
				return responseFormatJSON
			case strings.EqualFold(name, "html"): // text/html
				return responseFormatHTML
			case strings.EqualFold(name, "plain"): // text/plain
				return responseFormatPlainText
			}
		}

		return responseFormatPlainText
	}()

	var b bytes.Buffer

	b.Grow(1024) //nolint:mnd

	const contentTypeHeader = "Content-Type"

	switch format {
	case responseFormatHTML:
		w.Header().Set(contentTypeHeader, "text/html; charset=utf-8")

		b.WriteString(`<!doctype html>
<html lang="en">
<head>
	<meta charset="utf-8"/>
	<meta name="robots" content="nofollow,noarchive,noindex">
	<meta name="viewport" content="width=device-width, initial-scale=1.0, viewport-fit=cover">
	<title>`)
		b.WriteString(http.StatusText(statusCode))
		b.WriteString(`</title>
	<style>
    :root {
			--color-bg: #efeffa;
			--color-text: #2b2b2b;
		}
		@media (prefers-color-scheme: dark) {
			:root {
				--color-bg: #2b2b2b;
				--color-text: #efeffa;
			}
		}
    html, body {
      margin: 0;
      padding: 0;
      overflow: hidden;
      overscroll-behavior: none;
    }
		body {
      height: 100dvh;
      display: flex;
      align-items: center;
      justify-content: center;
      padding: 0 clamp(1rem, 4vmin, 3rem);
			font-family: system-ui, -apple-system, "Segoe UI", Roboto, sans-serif;
			background-color: var(--color-bg);
			color: var(--color-text);
		}
		.container {
			text-align: center;
		}
	</style>
</head>
<body>
	<div class="container">
		<h1>WebHook: `)
		b.WriteString(http.StatusText(statusCode))
		b.WriteString(`</h1>
			<h3>`)
		b.WriteString(msg)
		b.WriteString(`</h3>
	</div>
</body>
</html>`)
	case responseFormatJSON:
		w.Header().Set(contentTypeHeader, "application/json; charset=utf-8")

		_ = json.NewEncoder(&b).Encode(map[string]string{ //nolint:errcheck,errchkjson
			"error":      http.StatusText(statusCode),
			"message":    msg,
			"powered_by": fmt.Sprint("WebhookTester/", appmeta.Version()),
		})
	case responseFormatPlainText:
		w.Header().Set(contentTypeHeader, "text/plain; charset=utf-8")

		b.WriteString("WebHook: ")
		b.WriteString(http.StatusText(statusCode))
		b.WriteRune('\n')
		b.WriteString(msg)
		b.WriteRune('\n')
	}

	w.Header().Set("Content-Length", strconv.Itoa(b.Len()))
	w.Header().Set("Server", fmt.Sprint("WebhookTester/", appmeta.Version()))
	w.Header().Set("X-Robots-Tag", "noindex, nofollow, nosnippet, noarchive")

	w.WriteHeader(statusCode)

	if _, err := w.Write(b.Bytes()); err != nil {
		log.Error("Failed to write the error response body",
			logger.Error(err),
			logger.Int("status_code", statusCode),
			logger.String("msg", msg),
		)
	}
}
