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
	sessionUUID := mux.Vars(r)["sessionUUID"]

	if deleted, err := h.storage.DeleteRequests(sessionUUID); err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		_, _ = w.Write(errors.NewServerError(http.StatusInternalServerError, err.Error()).ToJSON())

		return
	} else if !deleted {
		w.WriteHeader(http.StatusNotFound)
		_, _ = w.Write(errors.NewServerError(
			http.StatusNotFound,
			fmt.Sprintf("Requests for session with UUID %s was not found", sessionUUID),
		).ToJSON())

		return
	}

	if h.broadcaster != nil {
		go func(sessionUUID string) {
			_ = h.broadcaster.Publish(sessionUUID, broadcast.RequestsDeleted, "*")
		}(sessionUUID)
	}

	_ = h.json.NewEncoder(w).Encode(api.Status{
		Success: true,
	})
}
