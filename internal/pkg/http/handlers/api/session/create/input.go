package create

import (
	"errors"
	"net/http"
	"time"

	jsoniter "github.com/json-iterator/go"
)

type input struct {
	StatusCode      uint16
	ContentType     string
	Delay           time.Duration
	ResponseContent string
}

func ParseInput(body []byte) (*input, error) {
	var bodyPayload struct {
		StatusCode       *uint16 `json:"status_code"`    // optional
		ContentType      *string `json:"content_type"`   // optional
		ResponseDelaySec *uint8  `json:"response_delay"` // optional
		ResponseContent  *string `json:"response_body"`  // optional
	}

	if err := jsoniter.ConfigFastest.Unmarshal(body, &bodyPayload); err != nil {
		return nil, errors.New("cannot parse passed json")
	}

	// init with defaults
	var p = input{
		StatusCode:      http.StatusOK,
		ContentType:     "text/plain",
		Delay:           time.Duration(0),
		ResponseContent: "",
	}

	// override default values with passed if last is presents
	if v := bodyPayload.StatusCode; v != nil {
		p.StatusCode = *v
	}
	if v := bodyPayload.ContentType; v != nil {
		p.ContentType = *v
	}
	if v := bodyPayload.ResponseDelaySec; v != nil {
		p.Delay = time.Duration(*v) * time.Second
	}
	if v := bodyPayload.ResponseContent; v != nil {
		p.ResponseContent = *v
	}

	return &p, nil
}

func (in input) Validate() error {
	const (
		minStatusCode, maxStatusCode = 100, 530
		maxContentTypeLength         = 32
		maxResponseDelay             = time.Second * 30
		maxResponseContentLength     = 10240
	)

	if in.StatusCode < minStatusCode || in.StatusCode > maxStatusCode {
		return errors.New("wrong status code")
	}

	if len(in.ContentType) > maxContentTypeLength {
		return errors.New("content-type value is too large")
	}

	if in.Delay > maxResponseDelay {
		return errors.New("delay is too much")
	}

	if len(in.ResponseContent) > maxResponseContentLength {
		return errors.New("response content is too large")
	}

	return nil
}
