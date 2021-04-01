package create

import (
	"time"

	jsoniter "github.com/json-iterator/go"
)

type output struct {
	SessionUUID string
	Content     string
	StatusCode  uint16
	ContentType string
	Delay       time.Duration
	CreatedAt   time.Time
}

func (o output) ToJSON() ([]byte, error) {
	var res = struct {
		SessionUUID      string `json:"uuid"`
		ResponseSettings struct {
			Content       string `json:"content"`
			ContentType   string `json:"content_type"`
			Code          uint16 `json:"code"`
			DelaySec      uint8  `json:"delay_sec"`
			CreatedAtUnix int64  `json:"created_at_unix"`
		} `json:"response"`
	}{}

	res.SessionUUID = o.SessionUUID
	res.ResponseSettings.Content = o.Content
	res.ResponseSettings.ContentType = o.ContentType
	res.ResponseSettings.Code = o.StatusCode
	res.ResponseSettings.DelaySec = uint8(o.Delay.Seconds())
	res.ResponseSettings.CreatedAtUnix = o.CreatedAt.Unix()

	return jsoniter.ConfigFastest.Marshal(res)
}
