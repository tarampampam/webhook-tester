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
