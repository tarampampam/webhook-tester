// Package responder contains different HTTP responders ("sugared" functions for easy working with server responses).
package responder

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

// JSON writes passed model as a json-formatted string into writer.
func JSON(w http.ResponseWriter, model jsoner) {
	if name := "Content-Type"; w.Header().Get(name) == "" {
		w.Header().Set(name, "application/json; charset=utf-8")
	}

	content, err := model.ToJSON()
	if err != nil {
		fallback, _ := json.Marshal(struct { // fallback error struct (JSON)
			Success bool   `json:"success"`
			Message string `json:"message"`
		}{false, err.Error()})

		w.Header().Set("Content-Type", "application/json; charset=utf-8")
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

	if _, err = w.Write(content); err != nil {
		panic(err) // occurred something very bad
	}
}
