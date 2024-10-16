package encoding

type Encoder interface {
	// Encode marshals the given value into a byte slice.
	Encode(any) ([]byte, error)
}

type Decoder interface {
	// Decode unmarshal the given byte slice into the given value.
	Decode([]byte, any) error
}

type EncoderDecoder interface {
	Encoder
	Decoder
}
