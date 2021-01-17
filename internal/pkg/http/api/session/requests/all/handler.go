package all

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
	sessionUUID, sessionFound := mux.Vars(r)["sessionUUID"]
	if !sessionFound {
		errors.NewServerError(uint16(http.StatusInternalServerError), "cannot extract session UUID").RespondWithJSON(w)
		return
	}

	if session, err := h.storage.GetSession(sessionUUID); session == nil {
		if err != nil {
			errors.NewServerError(
				uint16(http.StatusInternalServerError), "cannot get session data: "+err.Error(),
			).RespondWithJSON(w)

			return
		}

		errors.NewServerError(
			uint16(http.StatusNotFound), fmt.Sprintf("session with UUID %s was not found", sessionUUID),
		).RespondWithJSON(w)

		return
	}

	data, err := h.storage.GetAllRequests(sessionUUID)
	if err != nil {
		errors.NewServerError(
			uint16(http.StatusInternalServerError), "cannot get requests data: "+err.Error(),
		).RespondWithJSON(w)

		return
	}

	var (
		encoder = h.json.NewEncoder(w)
		result  = make(api.StoredRequests, 0)
	)

	if data == nil {
		_ = encoder.Encode(result) // not 404 - just empty result
		return
	}

	for _, resp := range *data {
		result = append(result, api.StoredRequest{
			UUID:          resp.UUID,
			ClientAddr:    resp.Request.ClientAddr,
			Method:        resp.Request.Method,
			Content:       resp.Request.Content,
			Headers:       api.MapToHeaders(resp.Request.Headers).Sorted(),
			URI:           resp.Request.URI,
			CreatedAtUnix: resp.CreatedAtUnix,
		})
	}

	_ = encoder.Encode(result.Sorted())
}
