package stub

import (
	"net/http"
	"strings"

	"github.com/google/uuid"
)

func Handler(json string) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")

		_, _ = w.Write([]byte(strings.Replace(json, "%RAND_UUID%", uuid.New().String(), -1)))
	})
}
