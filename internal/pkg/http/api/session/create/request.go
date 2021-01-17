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

func (r *request) setDefaults() {
	if r.StatusCode == nil {
		var defaultStatusCode uint16 = 200
		r.StatusCode = &defaultStatusCode
	}

	if r.ContentType == nil {
		var defaultContentType string = "text/plain"
		r.ContentType = &defaultContentType
	}

	if r.ResponseDelaySec == nil {
		var defaultDelaySec uint8 = 0
		r.ResponseDelaySec = &defaultDelaySec
	}

	if r.ResponseContent == nil {
		var defaultContent string = ""
		r.ResponseContent = &defaultContent
	}
}
