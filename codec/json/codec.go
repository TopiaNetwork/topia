package json

import (
	"encoding/json"
	"io"
)

type MarshalJson struct{}

func (m *MarshalJson) Marshal(v interface{}) ([]byte, error) {
	return json.Marshal(v)
}

func (m *MarshalJson) Unmarshal(data []byte, v interface{}) error {
	return json.Unmarshal(data, v)
}

type EncoderJson struct {
	w           io.Writer
	jsonEncoder *json.Encoder
}

func NewEncoderJson(w io.Writer) *EncoderJson {
	jsonEncoder := json.NewEncoder(w)
	return &EncoderJson{
		w:           w,
		jsonEncoder: jsonEncoder,
	}
}

func (e *EncoderJson) Encode(v interface{}) error {
	return e.jsonEncoder.Encode(v)
}

func (e *EncoderJson) Reset(w io.Writer) {
	e.w = w
}

type DecoderJson struct {
	r           io.Reader
	jsonDecoder *json.Decoder
}

func NewDecoderJson(r io.Reader) *DecoderJson {
	return &DecoderJson{
		r:           r,
		jsonDecoder: json.NewDecoder(r),
	}
}

func (d *DecoderJson) Decode(v interface{}) error {
	return d.jsonDecoder.Decode(v)
}

func (d *DecoderJson) Reset(r io.Reader) {
	d.r = r
}
