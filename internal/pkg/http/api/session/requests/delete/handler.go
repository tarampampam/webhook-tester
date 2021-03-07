package delete

import (
	"fmt"
	api2 "github.com/tarampampam/webhook-tester/internal/pkg/http/handlers/api"
	"net/http"

	"github.com/gorilla/mux"
	jsoniter "github.com/json-iterator/go"
	"github.com/tarampampam/webhook-tester/internal/pkg/broadcast"
	"github.com/tarampampam/webhook-tester/internal/pkg/http/api"
	"github.com/tarampampam/webhook-tester/internal/pkg/storage"
)

type Handler struct {
	storage     storage.Storage
	broadcaster broadcaster
	json        jsoniter.API
}

type broadcaster interface {
	Publish(channel string, event broadcast.Event) error
}

func NewHandler(storage storage.Storage, broadcaster broadcaster) http.Handler {
	return &Handler{
		storage:     storage,
		broadcaster: broadcaster,
		json:        jsoniter.ConfigFastest,
	}
}

func (h *Handler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	sessionUUID, sessionFound := mux.Vars(r)["sessionUUID"]
	if !sessionFound {
		api2.Respond(w, api2.NewServerError(http.StatusInternalServerError, "cannot extract session UUID"))

		return
	}

	requestUUID, requestFound := mux.Vars(r)["requestUUID"]
	if !requestFound {
		api2.Respond(w, api2.NewServerError(http.StatusInternalServerError, "cannot extract request UUID"))

		return
	}

	if deleted, err := h.storage.DeleteRequest(sessionUUID, requestUUID); err != nil {
		api2.Respond(w, api2.NewServerError(http.StatusInternalServerError, err.Error()))

		return
	} else if !deleted {
		api2.Respond(w, api2.NewServerError(
			http.StatusNotFound, fmt.Sprintf("request with UUID %s was not found", requestUUID),
		))

		return
	}

	if h.broadcaster != nil {
		go func(sessionUUID, requestUUID string) {
			_ = h.broadcaster.Publish(sessionUUID, broadcast.NewRequestDeletedEvent(requestUUID))
		}(sessionUUID, requestUUID)
	}

	_ = h.json.NewEncoder(w).Encode(api.StatusResponse{
		Success: true,
	})
}
