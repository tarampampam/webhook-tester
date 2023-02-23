package webhook

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/labstack/echo/v4"

	"gh.tarampamp.am/webhook-tester/internal/config"
	"gh.tarampamp.am/webhook-tester/internal/pubsub"
	"gh.tarampamp.am/webhook-tester/internal/storage"
)

type webhookMetrics interface {
	IncrementProcessedWebHooks()
}

func New( //nolint:funlen,gocognit,gocyclo
	ctx context.Context,
	cfg config.Config,
	storage storage.Storage,
	pub pubsub.Publisher,
	metrics webhookMetrics,
) echo.MiddlewareFunc {
	var ignoreHeaderPrefixes = make([]string, len(cfg.IgnoreHeaderPrefixes))

	for i := 0; i < len(cfg.IgnoreHeaderPrefixes); i++ {
		ignoreHeaderPrefixes[i] = strings.ToUpper(strings.TrimSpace(cfg.IgnoreHeaderPrefixes[i])) // normalize each
	}

	return func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(c echo.Context) error {
			// extract the first URL segment
			if parts := strings.Split(strings.TrimLeft(c.Request().URL.RequestURI(), "/"), "/"); len(parts) > 0 { //nolint:nestif
				if sessionUuidStruct, uuidErr := uuid.Parse(parts[0]); uuidErr == nil { // and check it's a valid UUID
					var (
						sessionUuid = sessionUuidStruct.String()
						statusCode  int
					)

					c.Response().Header().Set("Access-Control-Allow-Origin", "*") // allow cross-original requests

					if len(parts) == 2 && len(parts[1]) == 3 { // try to extract second URL segment as status code
						if code, err := strconv.Atoi(parts[1]); err == nil && code >= 100 && code <= 599 { // and verify it too
							statusCode = code
						}
					}

					session, err := storage.GetSession(sessionUuid) // read current session info
					if err != nil {
						return respondWithError(c,
							http.StatusInternalServerError,
							err.Error(),
						)
					}

					if session == nil { // is the session exists?
						return respondWithError(c,
							http.StatusNotFound,
							fmt.Sprintf("The session with UUID %s was not found", sessionUuid),
						)
					}

					if statusCode == 0 {
						statusCode = int(session.Code())
					}

					var body []byte // for request body

					if rb := c.Request().Body; rb != nil {
						if body, err = io.ReadAll(rb); err != nil {
							return respondWithError(c, http.StatusInternalServerError, err.Error())
						}
					}

					if cfg.MaxRequestBodySize > 0 && len(body) > int(cfg.MaxRequestBodySize) { // check the body size
						return respondWithError(c,
							http.StatusInternalServerError,
							fmt.Sprintf("Request body is too large (current: %d, maximal: %d)", len(body), cfg.MaxRequestBodySize),
						)
					}

					var requestUuid string

					if requestUuid, err = storage.CreateRequest( // store request in a storage
						sessionUuid,
						c.RealIP(),
						c.Request().Method,
						c.Request().URL.RequestURI(),
						body,
						headersToStringsMap(c.Request().Header, ignoreHeaderPrefixes),
					); err != nil {
						return respondWithError(c,
							http.StatusInternalServerError,
							fmt.Sprintf("Request saving in storage failed: %s", err.Error()),
						)
					}

					go func() {
						metrics.IncrementProcessedWebHooks()

						_ = pub.Publish(sessionUuid, pubsub.NewRequestRegisteredEvent(requestUuid))
					}()

					if delay := session.Delay(); delay > 0 {
						timer := time.NewTimer(delay)

						select {
						case <-ctx.Done():
							timer.Stop()

							return respondWithError(c, http.StatusInternalServerError, "canceled")

						case <-timer.C:
							timer.Stop()
						}
					}

					return c.Blob(statusCode, session.ContentType(), session.Content())
				}
			}

			return next(c)
		}
	}
}

func respondWithError(c echo.Context, code int, msg string) error {
	var s strings.Builder

	s.Grow(1024) //nolint:gomnd

	s.WriteString(`<!doctype html>
<html lang="en">
<head>
    <meta charset="utf-8"/>
    <meta http-equiv="X-UA-Compatible" content="IE=edge"/>
    <meta name="viewport" content="width=device-width, initial-scale=1"/>
    <title>`)
	s.WriteString(http.StatusText(code))
	s.WriteString(`</title>
    <style>
        html,body {width:100%; height:100%; margin:0; padding:0; background-color: #2b2b2b; color: #efeffa}
        body {display:flex; justify-content:center; align-items:center; font-family:sans-serif}
        .container {text-align:center}
    </style>
</head>
<body>
    <div class="container">
        <h1>WebHook: `)
	s.WriteString(http.StatusText(code))
	s.WriteString(`</h1>
        <h3>`)
	s.WriteString(msg)
	s.WriteString(`</h3>
    </div>
</body>
</html>`)

	return c.HTML(code, s.String())
}

func headersToStringsMap(header http.Header, ignorePrefixes []string) map[string]string {
	result := make(map[string]string, len(header))

loop:
	for name, values := range header {
		if len(ignorePrefixes) > 0 {
			upperName := strings.ToUpper(name)

			for i := 0; i < len(ignorePrefixes); i++ {
				if strings.HasPrefix(upperName, ignorePrefixes[i]) {
					continue loop
				}
			}
		}

		result[name] = strings.Join(values, "; ")
	}

	return result
}
