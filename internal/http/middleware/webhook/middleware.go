package webhook

import (
	"context"
	"fmt"
	"io"
	"net"
	"net/http"
	"slices"
	"strconv"
	"strings"
	"time"

	"go.uber.org/zap"

	"gh.tarampamp.am/webhook-tester/v2/internal/http/openapi"
	"gh.tarampamp.am/webhook-tester/v2/internal/pubsub"
	"gh.tarampamp.am/webhook-tester/v2/internal/storage"
)

func New( //nolint:funlen,gocognit,gocyclo
	log *zap.Logger,
	db storage.Storage,
	pub pubsub.Publisher[pubsub.CapturedRequest],
	maxBodySize uint32,
) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			var sID, doIt = shouldCaptureRequest(r)
			if !doIt {
				next.ServeHTTP(w, r)

				return
			}

			var ctx = r.Context()

			// get the session from the storage
			sess, sErr := db.GetSession(ctx, sID)
			if sErr != nil {
				respondWithError(w, log, http.StatusInternalServerError, sErr.Error())

				return
			}

			// read the request body
			var body []byte

			if r.Body != nil {
				if b, err := io.ReadAll(r.Body); err == nil {
					body = b
				}
			}

			// check the request body size and respond with an error if it's too large
			if maxBodySize > 0 && uint32(len(body)) > maxBodySize { //nolint:gosec
				respondWithError(w, log,
					http.StatusRequestEntityTooLarge,
					fmt.Sprintf("The request body is too large (current: %d, max: %d)", len(body), maxBodySize),
				)

				return
			}

			// convert request headers into the storage format
			var rHeaders = make([]storage.HttpHeader, 0, len(r.Header))
			for name, value := range r.Header {
				rHeaders = append(rHeaders, storage.HttpHeader{Name: name, Value: strings.Join(value, "; ")})
			}

			// sort headers by name
			slices.SortFunc(rHeaders, func(i, j storage.HttpHeader) int { return strings.Compare(i.Name, j.Name) })

			// and save the request to the storage
			rID, rErr := db.NewRequest(ctx, sID, storage.Request{
				ClientAddr: extractRealIP(r),
				Method:     r.Method,
				Body:       body,
				Headers:    rHeaders,
				URL:        r.URL.RequestURI(),
			})
			if rErr != nil {
				respondWithError(w, log, http.StatusInternalServerError, rErr.Error())

				return
			}

			// publish the captured request to the pub/sub
			go func() {
				// read the actual data from the storage (the main point is the time of creation)
				captured, dbErr := db.GetRequest(ctx, sID, rID)
				if dbErr != nil {
					log.Error("failed to get a captured request", zap.Error(dbErr))

					return
				}

				var headers = make([]pubsub.HttpHeader, len(captured.Headers))
				for i, h := range captured.Headers {
					headers[i] = pubsub.HttpHeader{Name: h.Name, Value: h.Value}
				}

				if err := pub.Publish(ctx, sID, pubsub.CapturedRequest{
					ID:                 rID,
					ClientAddr:         captured.ClientAddr,
					Method:             captured.Method,
					Body:               captured.Body,
					Headers:            headers,
					URL:                captured.URL,
					CreatedAtUnixMilli: captured.CreatedAtUnixMilli,
				}); err != nil {
					log.Error("failed to publish a captured request", zap.Error(err))
				}
			}()

			// wait for the delay if it's set
			if sess.Delay > 0 {
				sleep(ctx, sess.Delay)
			}

			// set the header to allow CORS requests from any origin and method
			w.Header().Set("Access-Control-Allow-Origin", "*")
			w.Header().Set("Access-Control-Allow-Methods", "*")
			w.Header().Set("Access-Control-Allow-Headers", "*")

			// set the session headers
			for _, h := range sess.Headers {
				w.Header().Set(h.Name, h.Value)
			}

			// by default, use the status code from the session
			var statusCode = int(sess.Code)

			// extract requested status code from the request URL (it should be the last part)
			if parts := strings.Split(strings.Trim(r.URL.Path, "/"), "/"); len(parts) > 1 {
				// loop over parts slice from the end to the beginning
				for i := len(parts) - 1; i >= 0; i-- {
					if code, err := strconv.Atoi(parts[i]); err == nil && code >= 100 && code <= 599 {
						statusCode = code

						break
					}
				}
			}

			// set the status code
			w.WriteHeader(statusCode)

			// write the response body
			if _, err := w.Write(sess.ResponseBody); err != nil {
				log.Error("failed to write the response body", zap.Error(err))
			}
		})
	}
}

// shouldCaptureRequest checks if the request should be captured (the path starts with a valid UUID).
func shouldCaptureRequest(r *http.Request) (string, bool) {
	if r.URL == nil {
		return "", false
	}

	var clean = strings.TrimLeft(r.URL.Path, "/")

	if len(clean) >= openapi.UUIDLength && openapi.IsValidUUID(clean[:openapi.UUIDLength]) {
		return clean[:openapi.UUIDLength], true
	}

	return "", false
}

func respondWithError(w http.ResponseWriter, log *zap.Logger, code int, msg string) {
	var s strings.Builder

	s.Grow(1024) //nolint:mnd

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

	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.Header().Set("Content-Length", strconv.Itoa(s.Len()))
	w.WriteHeader(code)

	if _, err := w.Write([]byte(s.String())); err != nil {
		log.Error("failed to respond with an error", zap.Error(err), zap.Int("code", code), zap.String("msg", msg))
	}
}

// we will trust following HTTP headers for the real ip extracting (priority low -> high).
var trustHeaders = [...]string{"X-Forwarded-For", "X-Real-IP", "CF-Connecting-IP"} //nolint:gochecknoglobals

func extractRealIP(r *http.Request) string {
	var ip string

	for _, name := range trustHeaders {
		if value := r.Header.Get(name); value != "" {
			// `X-Forwarded-For` can be `10.0.0.1, 10.0.0.2, 10.0.0.3`
			if strings.Contains(value, ",") {
				parts := strings.Split(value, ",")

				if len(parts) > 0 {
					ip = strings.TrimSpace(parts[0])
				}
			} else {
				ip = strings.TrimSpace(value)
			}
		}
	}

	if net.ParseIP(ip) != nil {
		return ip
	}

	return strings.Split(r.RemoteAddr, ":")[0]
}

func sleep(ctx context.Context, d time.Duration) {
	var timer = time.NewTimer(d)
	defer timer.Stop()

	select {
	case <-ctx.Done():
		return
	case <-timer.C:
	}
}
