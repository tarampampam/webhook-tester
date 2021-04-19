package all

import (
	"encoding/base64"
	"fmt"
	"net/http"
	"sort"

	"github.com/tarampampam/webhook-tester/internal/pkg/http/handlers/api"
	"github.com/tarampampam/webhook-tester/internal/pkg/http/responder"
	"github.com/tarampampam/webhook-tester/internal/pkg/storage"

	"github.com/gorilla/mux"
)

func NewHandler(storage storage.Storage) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		sessionUUID, sessionFound := mux.Vars(r)["sessionUUID"]
		if !sessionFound {
			responder.JSON(w, api.NewServerError(http.StatusInternalServerError, "cannot extract session UUID"))

			return
		}

		if session, err := storage.GetSession(sessionUUID); session == nil {
			if err != nil {
				responder.JSON(w, api.NewServerError(
					http.StatusInternalServerError, "cannot read session data: "+err.Error(),
				))

				return
			}

			responder.JSON(w, api.NewServerError(
				http.StatusNotFound, fmt.Sprintf("session with UUID %s was not found", sessionUUID),
			))

			return
		}

		requests, err := storage.GetAllRequests(sessionUUID)
		if err != nil {
			responder.JSON(w, api.NewServerError(
				http.StatusInternalServerError, "cannot get requests data: "+err.Error(),
			))

			return
		}

		var result = make(api.SessionRequests, 0, len(requests))

		for i := 0; i < len(requests); i++ {
			var (
				headersMap = requests[i].Headers()
				headers    = make([]api.SessionRequestHeader, 0, len(headersMap))
			)

			for name, value := range headersMap {
				headers = append(headers, api.SessionRequestHeader{Name: name, Value: value})
			}

			sort.SliceStable(headers, func(j, k int) bool { return headers[j].Name < headers[k].Name })

			result = append(result, api.SessionRequest{
				UUID:          requests[i].UUID(),
				ClientAddr:    requests[i].ClientAddr(),
				Method:        requests[i].Method(),
				ContentBase64: base64.StdEncoding.EncodeToString(requests[i].Content()),
				Headers:       headers,
				URI:           requests[i].URI(),
				CreatedAtUnix: requests[i].CreatedAt().Unix(),
			})
		}

		responder.JSON(w, result)
	}
}
