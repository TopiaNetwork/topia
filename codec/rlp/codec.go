package rlp

import (
	"io"

	"github.com/ethereum/go-ethereum/rlp"
)

type MarshalRlp struct{}

func (m *MarshalRlp) Marshal(v interface{}) ([]byte, error) {
	return rlp.EncodeToBytes(v)
}

func (m *MarshalRlp) Unmarshal(data []byte, v interface{}) error {
	return rlp.DecodeBytes(data, v)
}

type EncoderRlp struct {
	w io.Writer
}

func NewEncoderRlp(w io.Writer) *EncoderRlp {
	return &EncoderRlp{
		w: w,
	}
}

func (e *EncoderRlp) Encode(v interface{}) error {
	return rlp.Encode(e.w, v)
}

func (e *EncoderRlp) Reset(w io.Writer) {
	e.w = w
}

type DecoderRlp struct {
	r io.Reader
}

func NewDecoderRlp(r io.Reader) *DecoderRlp {
	return &DecoderRlp{
		r: r,
	}
}

func (d *DecoderRlp) Decode(v interface{}) error {
	return rlp.Decode(d.r, v)
}

func (d *DecoderRlp) Reset(r io.Reader) {
	d.r = r
}
