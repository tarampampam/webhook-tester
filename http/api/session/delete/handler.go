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
	sessionUUID := mux.Vars(r)["sessionUUID"]

	// delete session
	if result, err := h.storage.DeleteSession(sessionUUID); err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		_, _ = w.Write(errors.NewServerError(http.StatusInternalServerError, err.Error()).ToJSON())

		return
	} else if !result {
		w.WriteHeader(http.StatusNotFound)
		_, _ = w.Write(errors.NewServerError(
			http.StatusNotFound,
			fmt.Sprintf("Session with UUID %s was not found", sessionUUID),
		).ToJSON())

		return
	}

	// and recorded session requests
	if _, err := h.storage.DeleteRequests(sessionUUID); err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		_, _ = w.Write(errors.NewServerError(http.StatusInternalServerError, err.Error()).ToJSON())

		return
	}

	_ = h.json.NewEncoder(w).Encode(api.Status{
		Success: true,
	})
}
