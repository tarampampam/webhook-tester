package delete

import (
	"fmt"
	"net/http"
	"webhook-tester/http/api"
	"webhook-tester/http/errors"
	"webhook-tester/storage"

	"github.com/gorilla/mux"
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
	sessionUUID, sessionFound := mux.Vars(r)["sessionUUID"]
	if !sessionFound {
		errors.NewServerError(uint16(http.StatusInternalServerError), "cannot extract session UUID").RespondWithJSON(w)
		return
	}

	// delete session
	if result, err := h.storage.DeleteSession(sessionUUID); err != nil {
		errors.NewServerError(uint16(http.StatusInternalServerError), err.Error()).RespondWithJSON(w)
		return
	} else if !result {
		errors.NewServerError(
			uint16(http.StatusNotFound), fmt.Sprintf("session with UUID %s was not found", sessionUUID),
		).RespondWithJSON(w)
		return
	}

	// and recorded session requests
	if _, err := h.storage.DeleteRequests(sessionUUID); err != nil {
		errors.NewServerError(uint16(http.StatusInternalServerError), err.Error()).RespondWithJSON(w)
		return
	}

	_ = h.json.NewEncoder(w).Encode(api.StatusResponse{
		Success: true,
	})
}
