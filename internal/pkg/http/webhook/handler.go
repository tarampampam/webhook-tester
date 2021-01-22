package webhook

import (
	"context"
	"fmt"
	"io/ioutil"
	"net"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/gorilla/mux"
	"github.com/tarampampam/webhook-tester/internal/pkg/broadcast"
	"github.com/tarampampam/webhook-tester/internal/pkg/config"
	"github.com/tarampampam/webhook-tester/internal/pkg/http/errors"
	"github.com/tarampampam/webhook-tester/internal/pkg/storage"
)

type broadcaster interface {
	Publish(ch string, e broadcast.Event) error
}

const maxBodyLength = 64 * 1024 // 64 KiB // TODO make configurable?

type Handler struct {
	ctx     context.Context
	storage storage.Storage
	br      broadcaster

	ignoreHeaderPrefixes []string
}

func NewHandler(ctx context.Context, cfg config.Config, storage storage.Storage, br broadcaster) http.Handler {
	ignoreHeaders := make([]string, len(cfg.IgnoreHeaderPrefixes))
	for i := 0; i < len(cfg.IgnoreHeaderPrefixes); i++ {
		ignoreHeaders[i] = strings.ToUpper(strings.TrimSpace(cfg.IgnoreHeaderPrefixes[i]))
	}

	return &Handler{
		ctx:     ctx,
		storage: storage,
		br:      br,

		ignoreHeaderPrefixes: ignoreHeaders,
	}
}

func (h *Handler) ServeHTTP(w http.ResponseWriter, r *http.Request) { //nolint:funlen
	sUUID, ok := mux.Vars(r)["sessionUUID"] // extract session UUID from the request variables
	if !ok {
		h.respondWithError(w, http.StatusInternalServerError, "cannot extract session UUID")

		return
	}

	session, err := h.storage.GetSession(sUUID) // read current session info
	if err != nil {
		h.respondWithError(w, http.StatusInternalServerError, "session reading failed: "+err.Error())

		return
	}

	if session == nil { // session is exists?
		h.respondWithError(w, http.StatusNotFound, "session with UUID "+sUUID+" was not found")

		return
	}

	var body = make([]byte, 0) // for request body

	if r.Body != nil {
		if body, err = ioutil.ReadAll(r.Body); err != nil {
			h.respondWithError(w, http.StatusInternalServerError, err.Error())

			return
		}
	}

	if bl := len(body); bl > maxBodyLength { // check passed body size
		h.respondWithError(w,
			http.StatusInternalServerError,
			fmt.Sprintf("request body is too large (current: %d, maximal: %d)", bl, maxBodyLength),
		)

		return
	}

	var rUUID string

	// store request in a storage
	if rUUID, err = h.storage.CreateRequest(
		sUUID,
		h.getRealClientAddress(r),
		r.Method,
		string(body),
		r.RequestURI,
		h.headerToStringsMap(r.Header),
	); err != nil {
		h.respondWithError(w, http.StatusInternalServerError, "request saving in storage failed: "+err.Error())

		return
	}

	// broadcast an event "new request was registered successful"
	go func() { _ = h.br.Publish(sUUID, broadcast.NewRequestRegisteredEvent(rUUID)) }()

	if delay := session.Delay(); delay > 0 {
		timer := time.NewTimer(delay)
		defer timer.Stop()

		select {
		case <-h.ctx.Done():
			h.respondWithError(w, http.StatusInternalServerError, "canceled")

			return

		case <-timer.C:
		}
	}

	w.Header().Set("Content-Type", session.ContentType())
	w.WriteHeader(h.getRequiredHTTPCode(r, session))

	_, _ = w.Write([]byte(session.Content()))
}

func (h *Handler) respondWithError(w http.ResponseWriter, code int, msg string) {
	errors.NewServerError(code, msg).RespondWithJSON(w)
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
		for i := 0; i < len(h.ignoreHeaderPrefixes); i++ {
			if strings.HasPrefix(strings.ToUpper(name), h.ignoreHeaderPrefixes[i]) {
				continue loop
			}
		}

		result[name] = strings.Join(values, "; ")
	}

	return result
}

var trustHeaders = [...]string{"X-Forwarded-For", "X-Real-IP", "CF-Connecting-IP"} //nolint:gochecknoglobals

func (h *Handler) getRealClientAddress(r *http.Request) string {
	var ip string

	for _, name := range trustHeaders {
		if value := r.Header.Get(name); value != "" {
			// `X-Forwarded-For` can be `10.0.0.1, 10.0.0.2, 10.0.0.3`
			if strings.Contains(value, ",") {
				parts := strings.Split(value, ",")

				if len(parts) >= 1 {
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
