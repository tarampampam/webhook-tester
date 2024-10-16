package encoding

import "encoding/json"

type JSON struct{}

var _ EncoderDecoder = (*JSON)(nil) // ensure interface implementation

func (JSON) Encode(v any) ([]byte, error)    { return json.Marshal(v) }
func (JSON) Decode(data []byte, v any) error { return json.Unmarshal(data, v) }
