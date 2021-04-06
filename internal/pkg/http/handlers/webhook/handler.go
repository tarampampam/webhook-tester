package webhook

import (
	"context"
	"fmt"
	"io/ioutil"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/gorilla/mux"
	"github.com/tarampampam/webhook-tester/internal/pkg/config"
	"github.com/tarampampam/webhook-tester/internal/pkg/pubsub"
	"github.com/tarampampam/webhook-tester/internal/pkg/realip"
	"github.com/tarampampam/webhook-tester/internal/pkg/storage"
)

type metrics interface {
	IncrementProcessedWebHooks()
}

type Handler struct {
	ctx     context.Context
	storage storage.Storage
	pub     pubsub.Publisher
	m       metrics

	maxBodySize          uint32
	ignoreHeaderPrefixes []string
}

func NewHandler(
	ctx context.Context,
	cfg config.Config,
	storage storage.Storage,
	pub pubsub.Publisher,
	m metrics,
) *Handler {
	ignoreHeaders := make([]string, len(cfg.IgnoreHeaderPrefixes))
	for i := 0; i < len(cfg.IgnoreHeaderPrefixes); i++ {
		ignoreHeaders[i] = strings.ToUpper(strings.TrimSpace(cfg.IgnoreHeaderPrefixes[i]))
	}

	return &Handler{
		ctx:     ctx,
		storage: storage,
		pub:     pub,
		m:       m,

		maxBodySize:          cfg.MaxRequestBodySize,
		ignoreHeaderPrefixes: ignoreHeaders,
	}
}

func (h *Handler) ServeHTTP(w http.ResponseWriter, r *http.Request) { //nolint:funlen
	sUUID, ok := mux.Vars(r)["sessionUUID"] // extract session UUID from the request variables
	if !ok {
		h.error(w, http.StatusInternalServerError, "cannot extract session UUID")

		return
	}

	session, err := h.storage.GetSession(sUUID) // read current session info
	if err != nil {
		h.error(w, http.StatusInternalServerError, "session reading failed: "+err.Error())

		return
	}

	if session == nil { // session is exists?
		h.error(w, http.StatusNotFound, "session with UUID "+sUUID+" was not found")

		return
	}

	var body []byte // for request body

	if r.Body != nil {
		if body, err = ioutil.ReadAll(r.Body); err != nil {
			h.error(w, http.StatusInternalServerError, err.Error())

			return
		}
	} else {
		body = make([]byte, 0)
	}

	if h.maxBodySize > 0 && uint32(len(body)) > h.maxBodySize { // check passed body size
		h.error(w,
			http.StatusInternalServerError,
			fmt.Sprintf("request body is too large (current: %d, maximal: %d)", len(body), h.maxBodySize),
		)

		return
	}

	var rUUID string

	// store request in a storage
	if rUUID, err = h.storage.CreateRequest(
		sUUID,
		realip.FromHTTPRequest(r),
		r.Method,
		string(body),
		r.RequestURI,
		h.headerToStringsMap(r.Header),
	); err != nil {
		h.error(w, http.StatusInternalServerError, "request saving in storage failed: "+err.Error())

		return
	}

	go func() {
		h.m.IncrementProcessedWebHooks()

		_ = h.pub.Publish(sUUID, pubsub.NewRequestRegisteredEvent(rUUID))
	}()

	if delay := session.Delay(); delay > 0 {
		timer := time.NewTimer(delay)

		select {
		case <-h.ctx.Done():
			timer.Stop()
			h.error(w, http.StatusInternalServerError, "canceled")

			return

		case <-timer.C:
			timer.Stop()
		}
	}

	w.Header().Set("Content-Type", session.ContentType())
	w.WriteHeader(h.getRequiredHTTPCode(r, session))

	_, _ = w.Write([]byte(session.Content()))
}

func (h *Handler) error(w http.ResponseWriter, code int, msg string) {
	w.Header().Set("Content-Type", "text/html; charset=utf-8")

	w.WriteHeader(code)

	_, _ = w.Write([]byte(`<!doctype html>
<!--
  WebHook error: ` + msg + `
-->
<html lang="en">
<head>
    <meta charset="utf-8"/>
    <meta http-equiv="X-UA-Compatible" content="IE=edge"/>
    <meta name="viewport" content="width=device-width, initial-scale=1"/>
    <title>` + http.StatusText(code) + `</title>
    <style>
        html,body {width:100%; height:100%; margin:0; padding:0; background-color: #2b2b2b; color: #efeffa}
        body {display:flex; justify-content:center; align-items:center; font-family:sans-serif}
        .container {text-align:center}
    </style>
</head>
<body>
    <div class="container">
        <h1>WebHook: ` + http.StatusText(code) + `</h1>
        <h3>` + msg + `</h3>
    </div>
</body>
</html>`),
	)
}

func (h *Handler) getRequiredHTTPCode(r *http.Request, session storage.Session) int {
	// try to extract required status code from the request
	if statusCode, ok := mux.Vars(r)["statusCode"]; ok {
		if code, err := strconv.Atoi(statusCode); err == nil && code >= 100 && code <= 599 {
			return code
		}
	}

	return int(session.Code())
}

func (h *Handler) headerToStringsMap(header http.Header) map[string]string {
	result := make(map[string]string, len(header))

loop:
	for name, values := range header {
		if len(h.ignoreHeaderPrefixes) > 0 {
			upperName := strings.ToUpper(name)

			for i := 0; i < len(h.ignoreHeaderPrefixes); i++ {
				if strings.HasPrefix(upperName, h.ignoreHeaderPrefixes[i]) {
					continue loop
				}
			}
		}

		result[name] = strings.Join(values, "; ")
	}

	return result
}
