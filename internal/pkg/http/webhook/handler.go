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
	"github.com/tarampampam/webhook-tester/internal/pkg/http/errors"
	"github.com/tarampampam/webhook-tester/internal/pkg/settings"
	"github.com/tarampampam/webhook-tester/internal/pkg/storage"
)

const maxBodyLength = 8192

type Handler struct {
	appSettings *settings.AppSettings
	storage     storage.Storage
	broadcaster broadcast.Broadcaster
}

func NewHandler(set *settings.AppSettings, storage storage.Storage, br broadcast.Broadcaster) http.Handler {
	return &Handler{
		appSettings: set,
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

	requestData, creationErr := h.storage.CreateRequest(sessionUUID, &storage.Request{
		ClientAddr: h.getRealClientAddress(r),
		Method:     r.Method,
		Content:    string(body),
		Headers:    h.headerToStringsMap(r.Header),
		URI:        r.RequestURI,
	})

	if creationErr != nil {
		errors.NewServerError(
			uint16(http.StatusInternalServerError), "cannot put session data into storage: "+creationErr.Error(),
		).RespondWithJSON(w)

		return
	}

	if h.broadcaster != nil {
		go func(sessionUUID, requestUUID string) {
			_ = h.broadcaster.Publish(sessionUUID, broadcast.RequestRegistered, requestUUID)
		}(sessionUUID, requestData.UUID)
	}

	if delay := sessionData.WebHookResponse.DelaySec; delay > 0 {
		time.Sleep(time.Second * time.Duration(delay))
	}

	w.Header().Set("Content-Type", sessionData.WebHookResponse.ContentType)
	w.WriteHeader(h.getRequiredHTTPCode(r, sessionData))

	_, _ = w.Write([]byte(sessionData.WebHookResponse.Content))
}

func (h *Handler) getRequiredHTTPCode(r *http.Request, sessionData *storage.SessionData) (result int) {
	// try to extract required status code from the request
	if statusCode, codeFound := mux.Vars(r)["statusCode"]; codeFound {
		if code, err := strconv.Atoi(statusCode); err == nil {
			if sessionData.WebHookResponse.Code >= 100 && sessionData.WebHookResponse.Code <= 599 {
				result = code
			}
		}
	} else {
		result = int(sessionData.WebHookResponse.Code)
	}

	return
}

func (h *Handler) headerToStringsMap(header http.Header) map[string]string {
	result := make(map[string]string)

	shouldBeIgnored := make([]string, len(h.appSettings.IgnoreHeaderPrefixes))
	for i, value := range h.appSettings.IgnoreHeaderPrefixes {
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

func (h *Handler) getRealClientAddress(r *http.Request) string {
	var (
		trustHeaders = [...]string{"X-Forwarded-For", "X-Real-IP", "CF-Connecting-IP"}
		ip           string
	)

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
