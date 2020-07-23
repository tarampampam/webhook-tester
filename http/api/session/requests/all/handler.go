package all

import (
	"net/http"
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
		_, _ = w.Write(errors.NewServerError(uint16(http.StatusNotFound), "session was not found").ToJSON())

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

	if data == nil {
		_ = h.json.NewEncoder(w).Encode(response{}) // not 404 - just empty result
		return
	}

	var result = response{}

	for _, resp := range *data {
		result[resp.UUID] = record{
			ClientAddr:    resp.Request.ClientAddr,
			Method:        resp.Request.Method,
			Content:       resp.Request.Content,
			Headers:       resp.Request.Headers,
			URI:           resp.Request.URI,
			CreatedAtUnix: resp.CreatedAtUnix,
		}
	}

	_ = h.json.NewEncoder(w).Encode(result)
}
