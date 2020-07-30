package create

import (
	"io/ioutil"
	"net/http"
	"webhook-tester/http/errors"
	"webhook-tester/storage"

	jsoniter "github.com/json-iterator/go"
)

type Handler struct {
	storage storage.Storage
	json    jsoniter.API
}

func NewHandler(storage storage.Storage) http.Handler {
	return &Handler{
		storage: storage,
		json:    jsoniter.ConfigFastest,
	}
}

func (h *Handler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if r.Body == nil {
		h.respondWithError(w, http.StatusBadRequest, "empty request body")
		return
	}

	body, readErr := ioutil.ReadAll(r.Body)
	if readErr != nil {
		h.respondWithError(w, http.StatusInternalServerError, readErr.Error())
		return
	}

	var request = request{}

	if err := h.json.Unmarshal(body, &request); err != nil {
		h.respondWithError(w, http.StatusBadRequest, "cannot parse passed json")
		return
	}

	if err := request.validate(); err != nil {
		h.respondWithError(w, http.StatusBadRequest, "invalid value passed: "+err.Error())
		return
	}

	request.setDefaults()

	var webHookResp = &storage.WebHookResponse{
		Content:     *request.ResponseContent,
		Code:        *request.StatusCode,
		ContentType: *request.ContentType,
		DelaySec:    *request.ResponseDelaySec,
	}

	sessionData, sessionErr := h.storage.CreateSession(webHookResp)
	if sessionErr != nil {
		h.respondWithError(w, http.StatusInternalServerError, sessionErr.Error())
		return
	}

	w.WriteHeader(http.StatusOK)

	_ = h.json.NewEncoder(w).Encode(response{
		UUID: sessionData.UUID,
		ResponseSettings: responseSettings{
			Content:       sessionData.WebHookResponse.ContentType,
			Code:          sessionData.WebHookResponse.Code,
			ContentType:   sessionData.WebHookResponse.ContentType,
			DelaySec:      sessionData.WebHookResponse.DelaySec,
			CreatedAtUnix: sessionData.CreatedAtUnix,
		},
	})
}

func (h *Handler) respondWithError(w http.ResponseWriter, code int, message string) {
	w.WriteHeader(code)

	_, _ = w.Write(errors.NewServerError(uint16(code), message).ToJSON())
}
