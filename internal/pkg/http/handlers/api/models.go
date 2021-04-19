package api

import (
	jsoniter "github.com/json-iterator/go"
)

type ServerError struct {
	Success bool   `json:"success"`
	Code    int    `json:"code"`
	Message string `json:"message"`
}

// NewServerError creates new server error model.
func NewServerError(code int, message string) ServerError {
	return ServerError{Success: false, Code: code, Message: message}
}

// StatusCode returns HTTP status code for current model.
func (e ServerError) StatusCode() int { return e.Code }

// ToJSON returns the JSON encoding of current model.
func (e ServerError) ToJSON() ([]byte, error) { return jsoniter.ConfigFastest.Marshal(e) }

type (
	SessionRequest struct {
		UUID          string                 `json:"uuid"`
		ClientAddr    string                 `json:"client_address"`
		Method        string                 `json:"method"`
		ContentBase64 string                 `json:"content_base64"`
		Headers       []SessionRequestHeader `json:"headers"`
		URI           string                 `json:"url"`
		CreatedAtUnix int64                  `json:"created_at_unix"`
	}

	SessionRequestHeader struct {
		Name  string `json:"name"`
		Value string `json:"value"`
	}
)

func (sr SessionRequest) ToJSON() ([]byte, error) { return jsoniter.ConfigFastest.Marshal(sr) }

type SessionRequests []SessionRequest

func (sr SessionRequests) ToJSON() ([]byte, error) { return jsoniter.ConfigFastest.Marshal(sr) }
