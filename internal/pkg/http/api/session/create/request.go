package create

import "errors"

type request struct {
	StatusCode       *uint16 `json:"status_code"`    // optional
	ContentType      *string `json:"content_type"`   // optional
	ResponseDelaySec *uint8  `json:"response_delay"` // optional
	ResponseContent  *string `json:"response_body"`  // optional
}

func (r *request) validate() error {
	if r.StatusCode != nil && (*r.StatusCode < 100 || *r.StatusCode > 530) {
		return errors.New("wrong status code value")
	}

	if r.ContentType != nil && len(*r.ContentType) > 32 {
		return errors.New("content-type value is too long")
	}

	if r.ResponseDelaySec != nil && *r.ResponseDelaySec > 30 {
		return errors.New("delay is too much")
	}

	if r.ResponseContent != nil && len(*r.ResponseContent) > 10240 {
		return errors.New("response content is too long")
	}

	return nil
}

func (r *request) statusCode() uint16 {
	if r.StatusCode == nil {
		return 200 //nolint:gomnd // default value
	}

	return *r.StatusCode
}

func (r *request) contentType() string {
	if r.ContentType == nil {
		return "text/plain" // default value
	}

	return *r.ContentType
}

func (r *request) responseDelaySec() uint8 {
	if r.ResponseDelaySec == nil {
		return 0 // default value
	}

	return *r.ResponseDelaySec
}

func (r *request) responseContent() string {
	if r.ResponseContent == nil {
		return "" // default value
	}

	return *r.ResponseContent
}
