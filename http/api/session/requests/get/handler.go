package get

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
	requestUUID := mux.Vars(r)["requestUUID"]

	data, gettingErr := h.storage.GetRequest(sessionUUID, requestUUID)

	if gettingErr != nil {
		w.WriteHeader(http.StatusInternalServerError)
		_, _ = w.Write(errors.NewServerError(
			uint16(http.StatusInternalServerError), "cannot read request data: "+gettingErr.Error(),
		).ToJSON())

		return
	}

	if data == nil {
		w.WriteHeader(http.StatusNotFound)
		_, _ = w.Write(errors.NewServerError(uint16(http.StatusNotFound), "request was not found").ToJSON())

		return
	}

	_ = h.json.NewEncoder(w).Encode(response{
		ClientAddr:    data.Request.ClientAddr,
		Method:        data.Request.Method,
		Content:       data.Request.Content,
		Headers:       data.Request.Headers,
		URI:           data.Request.URI,
		CreatedAtUnix: data.CreatedAtUnix,
	})
}
