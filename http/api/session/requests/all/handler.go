package all

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

	if session, err := h.storage.GetSession(sessionUUID); session == nil {
		if err != nil {
			w.WriteHeader(http.StatusInternalServerError)
			_, _ = w.Write(errors.NewServerError(
				uint16(http.StatusInternalServerError), "cannot get session data: "+err.Error(),
			).ToJSON())

			return
		}

		w.WriteHeader(http.StatusNotFound)
		_, _ = w.Write(errors.NewServerError(
			uint16(http.StatusNotFound),
			fmt.Sprintf("session with UUID %s was not found", sessionUUID),
		).ToJSON())

		return
	}

	data, err := h.storage.GetAllRequests(sessionUUID)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		_, _ = w.Write(errors.NewServerError(
			uint16(http.StatusInternalServerError), "cannot get requests data: "+err.Error(),
		).ToJSON())

		return
	}

	var (
		encoder = h.json.NewEncoder(w)
		result  = make(api.Requests, 0)
	)

	if data == nil {
		_ = encoder.Encode(result) // not 404 - just empty result
		return
	}

	for _, resp := range *data {
		result = append(result, api.Request{
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
