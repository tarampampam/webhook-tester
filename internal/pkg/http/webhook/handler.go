package webhook

import (
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

const maxBodyLength = 8192

type Handler struct {
	cfg         config.Config
	storage     storage.Storage
	broadcaster broadcaster
}

type broadcaster interface {
	Publish(channel string, event broadcast.Event) error
}

func NewHandler(cfg config.Config, storage storage.Storage, br broadcaster) http.Handler {
	return &Handler{
		cfg:         cfg,
		storage:     storage,
		broadcaster: br,
	}
}

func (h *Handler) ServeHTTP(w http.ResponseWriter, r *http.Request) { //nolint:funlen
	sessionUUID, sessionFound := mux.Vars(r)["sessionUUID"]
	if !sessionFound {
		errors.NewServerError(uint16(http.StatusInternalServerError), "cannot extract session UUID").RespondWithJSON(w)
		return
	}

	sessionData, sessionErr := h.storage.GetSession(sessionUUID)

	if sessionErr != nil {
		errors.NewServerError(
			uint16(http.StatusInternalServerError), "cannot read session data from storage: "+sessionErr.Error(),
		).RespondWithJSON(w)

		return
	}

	if sessionData == nil {
		errors.NewServerError(
			uint16(http.StatusNotFound), fmt.Sprintf("session with UUID %s was not found", sessionUUID),
		).RespondWithJSON(w)

		return
	}

	var body []byte

	if r.Body != nil {
		b, readErr := ioutil.ReadAll(r.Body)
		if readErr != nil {
			errors.NewServerError(uint16(http.StatusInternalServerError), readErr.Error()).RespondWithJSON(w)
			return
		}

		body = b
	} else {
		body = []byte{}
	}

	if l := len(body); l > maxBodyLength {
		errors.NewServerError(
			uint16(http.StatusBadRequest),
			fmt.Sprintf("request body is too large (current: %d, maximal: %d)", l, maxBodyLength),
		).RespondWithJSON(w)

		return
	}

	requestUUID, creationErr := h.storage.CreateRequest(sessionUUID,
		h.getRealClientAddress(r),
		r.Method,
		string(body),
		r.RequestURI,
		h.headerToStringsMap(r.Header),
	)

	if creationErr != nil {
		errors.NewServerError(
			uint16(http.StatusInternalServerError), "cannot put session data into storage: "+creationErr.Error(),
		).RespondWithJSON(w)

		return
	}

	if h.broadcaster != nil {
		go func(sessionUUID, requestUUID string) {
			_ = h.broadcaster.Publish(sessionUUID, broadcast.NewRequestRegisteredEvent(requestUUID))
		}(sessionUUID, requestUUID)
	}

	if delay := sessionData.Delay(); delay > 0 {
		timer := time.NewTimer(delay)
		<-timer.C
		timer.Stop()
	}

	w.Header().Set("Content-Type", sessionData.ContentType())
	w.WriteHeader(h.getRequiredHTTPCode(r, sessionData))

	_, _ = w.Write([]byte(sessionData.Content()))
}

func (h *Handler) getRequiredHTTPCode(r *http.Request, sessionData storage.Session) (result int) {
	// try to extract required status code from the request
	if statusCode, codeFound := mux.Vars(r)["statusCode"]; codeFound {
		if code, err := strconv.Atoi(statusCode); err == nil {
			if sessionData.Code() >= 100 && sessionData.Code() <= 599 {
				result = code
			}
		}
	} else {
		result = int(sessionData.Code())
	}

	return
}

func (h *Handler) headerToStringsMap(header http.Header) map[string]string {
	result := make(map[string]string)

	shouldBeIgnored := make([]string, len(h.cfg.IgnoreHeaderPrefixes))
	for i, value := range h.cfg.IgnoreHeaderPrefixes {
		shouldBeIgnored[i] = strings.ToUpper(strings.TrimSpace(value))
	}

main:
	for name, values := range header {
		for _, ignore := range shouldBeIgnored {
			if strings.HasPrefix(strings.ToUpper(name), ignore) {
				continue main
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
