package create

import (
	"encoding/base64"
	"errors"
	"net/http"
	"time"
	"unicode/utf8"

	jsoniter "github.com/json-iterator/go"
)

type input struct {
	StatusCode      uint16
	ContentType     string
	Delay           time.Duration
	ResponseContent []byte
}

func parseInput(body []byte) (*input, error) {
	var bodyPayload struct {
		StatusCode            *uint16 `json:"status_code"`             // optional
		ContentType           *string `json:"content_type"`            // optional
		ResponseDelaySec      *uint8  `json:"response_delay"`          // optional
		ResponseContentBase64 *string `json:"response_content_base64"` // optional
	}

	if err := jsoniter.ConfigFastest.Unmarshal(body, &bodyPayload); err != nil {
		return nil, errors.New("cannot parse passed json")
	}

	// init with defaults
	var p = input{
		StatusCode:  http.StatusOK,
		ContentType: "text/plain",
		Delay:       time.Duration(0),
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

	if v := bodyPayload.ResponseContentBase64; v != nil {
		data, err := base64.StdEncoding.DecodeString(*v)
		if err != nil {
			return nil, errors.New("cannot decode response body (wrong base64)")
		}

		p.ResponseContent = data
	} else {
		p.ResponseContent = []byte{}
	}

	return &p, nil
}

func (in input) Validate() error {
	const (
		minStatusCode, maxStatusCode = 100, 530
		maxContentTypeLength         = 32
		maxResponseContentLength     = 10240
	)

	if in.StatusCode < minStatusCode || in.StatusCode > maxStatusCode {
		return errors.New("wrong status code")
	}

	if utf8.RuneCountInString(in.ContentType) > maxContentTypeLength {
		return errors.New("content-type value is too large")
	}

	if in.Delay > maxResponseDelay {
		return errors.New("delay is too much")
	}

	if utf8.RuneCount(in.ResponseContent) > maxResponseContentLength {
		return errors.New("response content is too large")
	}

	return nil
}
