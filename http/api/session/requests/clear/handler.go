package clear

import (
	"fmt"
	"net/http"
	"webhook-tester/broadcast"
	"webhook-tester/http/api"
	"webhook-tester/http/errors"
	"webhook-tester/storage"

	"github.com/gorilla/mux"
	jsoniter "github.com/json-iterator/go"
)

type Handler struct {
	storage     storage.Storage
	broadcaster broadcast.Broadcaster
	json        jsoniter.API
}

func NewHandler(storage storage.Storage, broadcaster broadcast.Broadcaster) http.Handler {
	return &Handler{
		storage:     storage,
		broadcaster: broadcaster,
		json:        jsoniter.ConfigFastest,
	}
}

func (h *Handler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	sessionUUID, sessionFound := mux.Vars(r)["sessionUUID"]
	if !sessionFound {
		errors.NewServerError(uint16(http.StatusInternalServerError), "cannot extract session UUID").RespondWithJSON(w)
		return
	}

	if deleted, err := h.storage.DeleteRequests(sessionUUID); err != nil {
		errors.NewServerError(uint16(http.StatusInternalServerError), err.Error()).RespondWithJSON(w)
		return
	} else if !deleted {
		errors.NewServerError(
			uint16(http.StatusNotFound), fmt.Sprintf("requests for session with UUID %s was not found", sessionUUID),
		).RespondWithJSON(w)

		return
	}

	if h.broadcaster != nil {
		go func(sessionUUID string) {
			_ = h.broadcaster.Publish(sessionUUID, broadcast.RequestsDeleted, "*")
		}(sessionUUID)
	}

	_ = h.json.NewEncoder(w).Encode(api.StatusResponse{
		Success: true,
	})
}
