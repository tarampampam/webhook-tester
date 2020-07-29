package webhook

import (
	"fmt"
	"io/ioutil"
	"net"
	"net/http"
	"strconv"
	"strings"
	"time"
	"webhook-tester/broadcast"
	"webhook-tester/http/errors"
	"webhook-tester/storage"

	"github.com/gorilla/mux"
)

const maxBodyLength = 8192

type Handler struct {
	storage     storage.Storage
	broadcaster broadcast.Broadcaster
}

func NewHandler(storage storage.Storage, broadcaster broadcast.Broadcaster) http.Handler {
	return &Handler{
		storage:     storage,
		broadcaster: broadcaster,
	}
}

func (h *Handler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	sessionUUID := mux.Vars(r)["sessionUUID"]

	sessionData, sessionErr := h.storage.GetSession(sessionUUID)

	if sessionErr != nil {
		h.respondWithError(w, http.StatusInternalServerError, "cannot read session data from storage: "+sessionErr.Error())
		return
	}

	if sessionData == nil {
		h.respondWithError(w, http.StatusNotFound, fmt.Sprintf("session with UUID %s was not found", sessionUUID))
		return
	}

	body, readErr := ioutil.ReadAll(r.Body)
	if readErr != nil {
		h.respondWithError(w, http.StatusInternalServerError, readErr.Error())
		return
	}

	if l := len(body); l > maxBodyLength {
		h.respondWithError(w,
			http.StatusBadRequest,
			fmt.Sprintf("request body is too large (current: %d, maximal: %d)", l, maxBodyLength),
		)

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
		h.respondWithError(w, http.StatusNotFound, "cannot put request data into storage: "+creationErr.Error())
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

	w.WriteHeader(h.getRequiredHTTPCode(r, sessionData))
	w.Header().Set("Content-Type", sessionData.WebHookResponse.ContentType)

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

	for name, values := range header {
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

func (h *Handler) respondWithError(w http.ResponseWriter, code int, message string) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(code)

	_, _ = w.Write(errors.NewServerError(uint16(code), message).ToJSON())
}
