package create

import (
	"encoding/base64"
	"time"

	jsoniter "github.com/json-iterator/go"
)

type output struct {
	SessionUUID string
	Content     []byte
	StatusCode  uint16
	ContentType string
	Delay       time.Duration
	CreatedAt   time.Time
}

func (o output) ToJSON() ([]byte, error) {
	var res = struct {
		SessionUUID      string `json:"uuid"`
		ResponseSettings struct {
			ContentBase64 string `json:"content_base64"`
			ContentType   string `json:"content_type"`
			Code          uint16 `json:"code"`
			DelaySec      uint8  `json:"delay_sec"`
		} `json:"response"`
		CreatedAtUnix int64 `json:"created_at_unix"`
	}{}

	res.SessionUUID = o.SessionUUID
	res.ResponseSettings.ContentBase64 = base64.StdEncoding.EncodeToString(o.Content)
	res.ResponseSettings.ContentType = o.ContentType
	res.ResponseSettings.Code = o.StatusCode
	res.ResponseSettings.DelaySec = uint8(o.Delay.Seconds())
	res.CreatedAtUnix = o.CreatedAt.Unix()

	return jsoniter.ConfigFastest.Marshal(res)
}
