package delete

import (
	"fmt"
	"net/http"

	api2 "github.com/tarampampam/webhook-tester/internal/pkg/http/handlers/api"

	"github.com/gorilla/mux"
	jsoniter "github.com/json-iterator/go"
	"github.com/tarampampam/webhook-tester/internal/pkg/http/api"
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
	sessionUUID, sessionFound := mux.Vars(r)["sessionUUID"]
	if !sessionFound {
		api2.Respond(w, api2.NewServerError(
			http.StatusInternalServerError, "cannot extract session UUID",
		))

		return
	}

	// delete session
	if result, err := h.storage.DeleteSession(sessionUUID); err != nil {
		api2.Respond(w, api2.NewServerError(
			http.StatusInternalServerError, err.Error(),
		))

		return
	} else if !result {
		api2.Respond(w, api2.NewServerError(
			http.StatusNotFound, fmt.Sprintf("session with UUID %s was not found", sessionUUID),
		))

		return
	}

	// and recorded session requests
	if _, err := h.storage.DeleteRequests(sessionUUID); err != nil { // TODO delete requests first and ignore error?
		api2.Respond(w, api2.NewServerError(http.StatusInternalServerError, err.Error()))

		return
	}

	_ = h.json.NewEncoder(w).Encode(api.StatusResponse{
		Success: true,
	})
}
