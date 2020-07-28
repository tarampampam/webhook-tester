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
	body, readErr := ioutil.ReadAll(r.Body)
	if readErr != nil {
		w.WriteHeader(http.StatusInternalServerError)
		_, _ = w.Write(errors.NewServerError(http.StatusInternalServerError, readErr.Error()).ToJSON())

		return
	}

	var request = request{}

	if err := h.json.Unmarshal(body, &request); err != nil {
		w.WriteHeader(http.StatusBadRequest)
		_, _ = w.Write(errors.NewServerError(http.StatusBadRequest, err.Error()).ToJSON())

		return
	}

	if err := request.validate(); err != nil {
		w.WriteHeader(http.StatusBadRequest)
		_, _ = w.Write(errors.NewServerError(http.StatusBadRequest, err.Error()).ToJSON())

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
		w.WriteHeader(http.StatusInternalServerError)
		_, _ = w.Write(errors.NewServerError(http.StatusInternalServerError, sessionErr.Error()).ToJSON())

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
