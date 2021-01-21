package create

import (
	"io/ioutil"
	"net/http"
	"time"

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

	var req = request{}

	if err := h.json.Unmarshal(body, &req); err != nil {
		errors.NewServerError(uint16(http.StatusBadRequest), "cannot parse passed json").RespondWithJSON(w)
		return
	}

	if err := req.validate(); err != nil {
		errors.NewServerError(uint16(http.StatusBadRequest), "invalid value passed: "+err.Error()).RespondWithJSON(w)
		return
	}

	sessionUUID, sessionErr := h.storage.CreateSession(
		req.responseContent(),
		req.statusCode(),
		req.contentType(),
		time.Second*time.Duration(req.responseDelaySec()),
	)
	if sessionErr != nil {
		errors.NewServerError(uint16(http.StatusInternalServerError), sessionErr.Error()).RespondWithJSON(w)
		return
	}

	w.WriteHeader(http.StatusOK)

	_ = h.json.NewEncoder(w).Encode(response{
		UUID: sessionUUID,
		ResponseSettings: responseSettings{
			Content:       req.responseContent(),
			Code:          req.statusCode(),
			ContentType:   req.contentType(),
			DelaySec:      req.responseDelaySec(),
			CreatedAtUnix: time.Now().Unix(),
		},
	})
}
