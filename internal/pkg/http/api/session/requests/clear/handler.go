package clear

import (
	"fmt"
	"net/http"

	"github.com/gorilla/mux"
	jsoniter "github.com/json-iterator/go"
	"github.com/tarampampam/webhook-tester/internal/pkg/broadcast"
	"github.com/tarampampam/webhook-tester/internal/pkg/http/api"
	"github.com/tarampampam/webhook-tester/internal/pkg/http/errors"
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
		errors.NewServerError(http.StatusInternalServerError, "cannot extract session UUID").RespondWithJSON(w)
		return
	}

	if deleted, err := h.storage.DeleteRequests(sessionUUID); err != nil {
		errors.NewServerError(http.StatusInternalServerError, err.Error()).RespondWithJSON(w)
		return
	} else if !deleted {
		errors.NewServerError(
			http.StatusNotFound, fmt.Sprintf("requests for session with UUID %s was not found", sessionUUID),
		).RespondWithJSON(w)

		return
	}

	if h.broadcaster != nil {
		go func(sessionUUID string) {
			_ = h.broadcaster.Publish(sessionUUID, broadcast.NewAllRequestsDeletedEvent())
		}(sessionUUID)
	}

	_ = h.json.NewEncoder(w).Encode(api.StatusResponse{
		Success: true,
	})
}
