package create

import (
	"io/ioutil"
	"net/http"

	jsoniter "github.com/json-iterator/go"
	"github.com/tarampampam/webhook-tester/internal/pkg/http/errors"
	"github.com/tarampampam/webhook-tester/internal/pkg/storage"
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
		errors.NewServerError(uint16(http.StatusBadRequest), "empty request body").RespondWithJSON(w)
		return
	}

	body, readErr := ioutil.ReadAll(r.Body)
	if readErr != nil {
		errors.NewServerError(uint16(http.StatusInternalServerError), readErr.Error()).RespondWithJSON(w)
		return
	}

	var request = request{}

	if err := h.json.Unmarshal(body, &request); err != nil {
		errors.NewServerError(uint16(http.StatusBadRequest), "cannot parse passed json").RespondWithJSON(w)
		return
	}

	if err := request.validate(); err != nil {
		errors.NewServerError(uint16(http.StatusBadRequest), "invalid value passed: "+err.Error()).RespondWithJSON(w)
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
		errors.NewServerError(uint16(http.StatusInternalServerError), sessionErr.Error()).RespondWithJSON(w)
		return
	}

	w.WriteHeader(http.StatusOK)

	_ = h.json.NewEncoder(w).Encode(response{
		UUID: sessionData.UUID,
		ResponseSettings: responseSettings{
			Content:       sessionData.WebHookResponse.Content,
			Code:          sessionData.WebHookResponse.Code,
			ContentType:   sessionData.WebHookResponse.ContentType,
			DelaySec:      sessionData.WebHookResponse.DelaySec,
			CreatedAtUnix: sessionData.CreatedAtUnix,
		},
	})
}
