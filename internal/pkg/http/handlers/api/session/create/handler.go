package create

import (
	"io/ioutil"
	"net/http"
	"time"

	"github.com/tarampampam/webhook-tester/internal/pkg/http/handlers/api"

	"github.com/tarampampam/webhook-tester/internal/pkg/storage"
)

func NewHandler(storage storage.Storage) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Body == nil {
			api.Respond(w, api.NewServerError(http.StatusBadRequest, "empty request body"))

			return
		}

		body, readingErr := ioutil.ReadAll(r.Body)
		if readingErr != nil {
			api.Respond(w, api.NewServerError(http.StatusInternalServerError, readingErr.Error()))

			return
		}

		payload, parsingErr := ParseInput(body)
		if parsingErr != nil {
			api.Respond(w, api.NewServerError(http.StatusBadRequest, parsingErr.Error()))

			return
		}

		if err := payload.Validate(); err != nil {
			api.Respond(w, api.NewServerError(http.StatusBadRequest, "wrong request: "+err.Error()))

			return
		}

		uuid, savingErr := storage.CreateSession(
			payload.ResponseContent,
			payload.StatusCode,
			payload.ContentType,
			payload.Delay,
		)
		if savingErr != nil {
			api.Respond(w, api.NewServerError(http.StatusInternalServerError, savingErr.Error()))

			return
		}

		api.Respond(w, output{
			SessionUUID: uuid,
			Content:     payload.ResponseContent,
			StatusCode:  payload.StatusCode,
			ContentType: payload.ContentType,
			Delay:       payload.Delay,
			CreatedAt:   time.Now(),
		})
	}
}
