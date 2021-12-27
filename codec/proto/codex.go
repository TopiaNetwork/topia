package proto

import (
	"bytes"
	"errors"
	"fmt"
	"io"

	"github.com/gogo/protobuf/proto"
)

type GogoProtoObj interface {
	proto.Marshaler
	proto.Unmarshaler
	proto.Message
}

type MarshalProto struct{}

func (m *MarshalProto) Marshal(obj interface{}) ([]byte, error) {
	if msg, ok := obj.(GogoProtoObj); ok {
		return msg.Marshal()
	}
	return nil, fmt.Errorf("can not serialize the object %T", obj)
}

func (m *MarshalProto) Unmarshal(buff []byte, obj interface{}) error {
	if msg, ok := obj.(GogoProtoObj); ok {
		msg.Reset()
		return msg.Unmarshal(buff)
	}

	return fmt.Errorf("%T obj does not implement proto.Message", obj)
}

type EncoderProto struct {
	pBuf *proto.Buffer
	w    io.Writer
}

func NewEncoderProto(w io.Writer) *EncoderProto {
	return &EncoderProto{
		pBuf: proto.NewBuffer(nil),
		w:    w,
	}
}

// Encode marshals a proto.Message and writes it to an io.Writer
func (e *EncoderProto) Encode(v interface{}) error {
	msg, ok := v.(proto.Message)
	if !ok {
		return errors.New("Cannot encode struct that doesn't implement proto.Message")
	}

	var err error

	// TODO: pipe marshal directly to writer
	if err = e.pBuf.Marshal(msg); err != nil {
		return err
	}

	if _, err = e.w.Write(e.pBuf.Bytes()); err != nil {
		return err
	}
	return nil
}

func (e *EncoderProto) Reset(w io.Writer) {
	e.pBuf.Reset()
	e.w = w
}

type DecoderProto struct {
	pBuf *proto.Buffer
	bBuf *bytes.Buffer
	r    io.Reader
}

func NewDecoderProto(r io.Reader) *DecoderProto {
	return &DecoderProto{
		pBuf: proto.NewBuffer(nil),
		bBuf: &bytes.Buffer{},
		r:    r,
	}
}

func (d *DecoderProto) Decode(v interface{}) error {
	msg, ok := v.(proto.Message)
	if !ok {
		return errors.New("Cannot decode into struct that doesn't implement proto.Message")
	}

	var err error

	// TODO: pipe reader directly to proto.Buffer
	if _, err = d.bBuf.ReadFrom(d.r); err != nil {
		return err
	}
	d.pBuf.SetBuf(d.bBuf.Bytes())

	if err = d.pBuf.Unmarshal(msg); err != nil {
		return err
	}
	return nil
}

// Reset stores the new reader and resets its bytes.Buffer and proto.Buffer
func (d *DecoderProto) Reset(r io.Reader) {
	d.pBuf.Reset()
	d.bBuf.Reset()
	d.r = r
}
