package get

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
	sessionUUID := mux.Vars(r)["sessionUUID"]
	requestUUID := mux.Vars(r)["requestUUID"]

	req, gettingErr := h.storage.GetRequest(sessionUUID, requestUUID)

	if gettingErr != nil {
		api2.Respond(w, api2.NewServerError(
			http.StatusInternalServerError, "cannot read request data: "+gettingErr.Error(),
		))

		return
	}

	if req == nil {
		api2.Respond(w, api2.NewServerError(
			http.StatusNotFound, fmt.Sprintf("request with UUID %s was not found", requestUUID),
		))

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
