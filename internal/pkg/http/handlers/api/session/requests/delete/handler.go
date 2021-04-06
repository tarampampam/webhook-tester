package delete

import (
	"fmt"
	"net/http"

	"github.com/gorilla/mux"
	jsoniter "github.com/json-iterator/go"
	"github.com/tarampampam/webhook-tester/internal/pkg/http/handlers/api"
	"github.com/tarampampam/webhook-tester/internal/pkg/http/responder"
	"github.com/tarampampam/webhook-tester/internal/pkg/pubsub"
	"github.com/tarampampam/webhook-tester/internal/pkg/storage"
)

func NewHandler(storage storage.Storage, pub pubsub.Publisher) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		sessionUUID, sessionFound := mux.Vars(r)["sessionUUID"]
		if !sessionFound {
			responder.JSON(w, api.NewServerError(http.StatusInternalServerError, "cannot extract session UUID"))

			return
		}

		requestUUID, requestFound := mux.Vars(r)["requestUUID"]
		if !requestFound {
			responder.JSON(w, api.NewServerError(http.StatusInternalServerError, "cannot extract request UUID"))

			return
		}

		if deleted, err := storage.DeleteRequest(sessionUUID, requestUUID); err != nil {
			responder.JSON(w, api.NewServerError(http.StatusInternalServerError, err.Error()))

			return
		} else if !deleted {
			responder.JSON(w, api.NewServerError(
				http.StatusNotFound, fmt.Sprintf("request with UUID %s was not found", requestUUID),
			))

			return
		}

		go func() { _ = pub.Publish(sessionUUID, pubsub.NewRequestDeletedEvent(requestUUID)) }()

		responder.JSON(w, output{Success: true})
	}
}

type output struct {
	Success bool `json:"success"`
}

func (o output) ToJSON() ([]byte, error) { return jsoniter.ConfigFastest.Marshal(o) }
