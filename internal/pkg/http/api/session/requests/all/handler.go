package all

import (
	"fmt"
	api2 "github.com/tarampampam/webhook-tester/internal/pkg/http/handlers/api"
	"net/http"

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
		api2.Respond(w, api2.NewServerError(http.StatusInternalServerError, "cannot extract session UUID"))

		return
	}

	if session, err := h.storage.GetSession(sessionUUID); session == nil {
		if err != nil {
			api2.Respond(w, api2.NewServerError(
				http.StatusInternalServerError, "cannot get session data: "+err.Error(),
			))

			return
		}

		api2.Respond(w, api2.NewServerError(
			http.StatusNotFound, fmt.Sprintf("session with UUID %s was not found", sessionUUID),
		))

		return
	}

	allReq, err := h.storage.GetAllRequests(sessionUUID)
	if err != nil {
		api2.Respond(w, api2.NewServerError(
			http.StatusInternalServerError, "cannot get requests data: "+err.Error(),
		))

		return
	}

	var (
		encoder = h.json.NewEncoder(w)
		result  = make(api.StoredRequests, 0)
	)

	if allReq == nil {
		_ = encoder.Encode(result) // not 404 - just empty result
		return
	}

	for _, req := range allReq {
		result = append(result, api.StoredRequest{
			UUID:          req.UUID(),
			ClientAddr:    req.ClientAddr(),
			Method:        req.Method(),
			Content:       req.Content(),
			Headers:       api.MapToHeaders(req.Headers()).Sorted(),
			URI:           req.URI(),
			CreatedAtUnix: req.CreatedAt().Unix(),
		})
	}

	_ = encoder.Encode(result.Sorted())
}
