package errors

import jsoniter "github.com/json-iterator/go"

type ServerError struct {
	Success bool   `json:"success"`
	Code    uint16 `json:"code"`
	Message string `json:"message"`
}

func NewServerError(code uint16, message string) *ServerError {
	return &ServerError{
		Success: false,
		Code:    code,
		Message: message,
	}
}

// Get error message.
func (e *ServerError) Error() string {
	return e.Message
}

func (e *ServerError) ToJSON() []byte {
	if marshaled, err := jsoniter.ConfigFastest.Marshal(e); err == nil {
		return marshaled
	}

	return []byte(`{"error cannot be converted into JSON representation"}`)
}
