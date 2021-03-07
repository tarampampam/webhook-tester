package api

import (
	"encoding/json"
	"net/http"
)

type httpStatusCoder interface {
	StatusCode() int
}

type jsoner interface {
	ToJSON() ([]byte, error)
}

func Respond(w http.ResponseWriter, model jsoner) {
	if name := "Content-Type"; w.Header().Get(name) == "" {
		w.Header().Set(name, "application/json")
	}

	content, err := model.ToJSON()
	if err != nil {
		fallback, _ := json.Marshal(struct { // fallback error struct (JSON)
			Success bool   `json:"success"`
			Message string `json:"message"`
		}{false, err.Error()})

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusInternalServerError)
		_, _ = w.Write(fallback)

		return
	}

	var code = http.StatusOK // default code

	if v, ok := model.(httpStatusCoder); ok {
		if statusCode := v.StatusCode(); statusCode != 0 {
			code = statusCode // override with model code
		}
	}

	w.WriteHeader(code)

	_, _ = w.Write(content)
}
