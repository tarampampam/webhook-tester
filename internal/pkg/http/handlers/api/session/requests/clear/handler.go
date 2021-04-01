package clear

import (
	"fmt"
	"net/http"

	"github.com/gorilla/mux"
	jsoniter "github.com/json-iterator/go"
	"github.com/tarampampam/webhook-tester/internal/pkg/broadcast"
	"github.com/tarampampam/webhook-tester/internal/pkg/http/handlers/api"
	"github.com/tarampampam/webhook-tester/internal/pkg/http/responder"
	"github.com/tarampampam/webhook-tester/internal/pkg/storage"
)

type broadcaster interface {
	Publish(channel string, event broadcast.Event) error
}

func NewHandler(storage storage.Storage, br broadcaster) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		sessionUUID, sessionFound := mux.Vars(r)["sessionUUID"]
		if !sessionFound {
			responder.JSON(w, api.NewServerError(http.StatusInternalServerError, "cannot extract session UUID"))

			return
		}

		if deleted, err := storage.DeleteRequests(sessionUUID); err != nil {
			responder.JSON(w, api.NewServerError(http.StatusInternalServerError, err.Error()))

			return
		} else if !deleted {
			responder.JSON(w, api.NewServerError(
				http.StatusNotFound, fmt.Sprintf("requests for session with UUID %s was not found", sessionUUID),
			))

			return
		}

		if br != nil {
			go func() { _ = br.Publish(sessionUUID, broadcast.NewAllRequestsDeletedEvent()) }()
		}

		responder.JSON(w, output{Success: true})
	}
}

type output struct {
	Success bool `json:"success"`
}

func (o output) ToJSON() ([]byte, error) { return jsoniter.ConfigFastest.Marshal(o) }
