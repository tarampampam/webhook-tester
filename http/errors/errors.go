package errors

type JSONError struct {
	Error   bool   `json:"error"`
	Code    uint16 `json:"code"`
	Message string `json:"message"`
}

func NewJSONError(code uint16, message string) *JSONError {
	return &JSONError{
		Error:   true,
		Code:    code,
		Message: message,
	}
}
