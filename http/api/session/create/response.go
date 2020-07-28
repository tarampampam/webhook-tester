package create

type responseSettings struct {
	Content       string `json:"content"`
	Code          uint16 `json:"code"`
	ContentType   string `json:"content_type"`
	DelaySec      uint8  `json:"delay_sec"`
	CreatedAtUnix int64  `json:"created_at_unix"`
}

type response struct {
	UUID             string           `json:"uuid"`
	ResponseSettings responseSettings `json:"response"`
}
