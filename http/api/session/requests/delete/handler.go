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
	requestUUID := mux.Vars(r)["requestUUID"]

	if deleted, err := h.storage.DeleteRequest(sessionUUID, requestUUID); err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		_, _ = w.Write(errors.NewServerError(http.StatusInternalServerError, err.Error()).ToJSON())

		return
	} else if !deleted {
		w.WriteHeader(http.StatusNotFound)
		_, _ = w.Write(errors.NewServerError(
			http.StatusNotFound,
			fmt.Sprintf("Request with UUID %s was not found", requestUUID),
		).ToJSON())

		return
	}

	_ = h.json.NewEncoder(w).Encode(api.Status{
		Success: true,
	})
}
