package get

import (
	"fmt"
	"net/http"

	"github.com/gorilla/mux"
	jsoniter "github.com/json-iterator/go"
	"github.com/tarampampam/webhook-tester/internal/pkg/http/api"
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
	sessionUUID := mux.Vars(r)["sessionUUID"]
	requestUUID := mux.Vars(r)["requestUUID"]

	req, gettingErr := h.storage.GetRequest(sessionUUID, requestUUID)

	if gettingErr != nil {
		w.WriteHeader(http.StatusInternalServerError)
		_, _ = w.Write(errors.NewServerError(
			http.StatusInternalServerError, "cannot read request data: "+gettingErr.Error(),
		).ToJSON())

		return
	}

	if req == nil {
		w.WriteHeader(http.StatusNotFound)
		_, _ = w.Write(errors.NewServerError(
			http.StatusNotFound,
			fmt.Sprintf("request with UUID %s was not found", requestUUID),
		).ToJSON())

		return
	}

	_ = h.json.NewEncoder(w).Encode(api.StoredRequest{
		UUID:          requestUUID,
		ClientAddr:    req.ClientAddr(),
		Method:        req.Method(),
		Content:       req.Content(),
		Headers:       api.MapToHeaders(req.Headers()).Sorted(),
		URI:           req.URI(),
		CreatedAtUnix: req.CreatedAt().Unix(),
	})
}
