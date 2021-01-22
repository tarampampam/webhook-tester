package errors

import (
	"net/http"

	jsoniter "github.com/json-iterator/go"
)

type serverError struct {
	Success bool   `json:"success"`
	Code    int    `json:"code"`
	Message string `json:"message"`
}

func NewServerError(code int, message string) *serverError {
	return &serverError{Success: false, Code: code, Message: message}
}

// Get error message.
func (e *serverError) Error() string { return e.Message }

func (e *serverError) ToJSON() []byte {
	if j, err := jsoniter.ConfigFastest.Marshal(e); err == nil {
		return j
	}

	return []byte(`{"error cannot be converted into JSON representation"}`) // fallback
}

func (e *serverError) RespondWithJSON(w http.ResponseWriter) {
	if k, v := "Content-Type", "application/json"; w.Header().Get(k) != v {
		w.Header().Set(k, v)
	}

	w.WriteHeader(e.Code)

	_, _ = w.Write(e.ToJSON())
}
