package codec

import (
	"fmt"
	"io"

	"github.com/TopiaNetwork/topia/codec/json"
	"github.com/TopiaNetwork/topia/codec/proto"
	"github.com/TopiaNetwork/topia/codec/rlp"
)

type CodecType byte

const (
	CodecType_Unknown = iota
	CodecType_JSON
	CodecType_PROTO
	CodecType_RLP
)

type Marshaler interface {
	Marshal(interface{}) ([]byte, error)

	Unmarshal([]byte, interface{}) error
}

type Encoder interface {
	Encode(interface{}) error
	Reset(w io.Writer)
}

type Decoder interface {
	Decode(interface{}) error
	Reset(r io.Reader)
}

func CreateMarshaler(codecType CodecType) Marshaler {
	switch codecType {
	case CodecType_JSON:
		return &json.MarshalJson{}
	case CodecType_PROTO:
		return &proto.MarshalProto{}
	case CodecType_RLP:
		return &rlp.MarshalRlp{}
	default:
		panic(fmt.Errorf("invalid codec type %d when CreateMarshaler", codecType).Error())
	}

	return nil
}

func CreateEncoder(codecType CodecType, w io.Writer) Encoder {
	switch codecType {
	case CodecType_JSON:
		return json.NewEncoderJson(w)
	case CodecType_PROTO:
		return proto.NewEncoderProto(w)
	case CodecType_RLP:
		return rlp.NewEncoderRlp(w)
	default:
		panic(fmt.Errorf("invalid codec type %d when CreateEncoder", codecType).Error())
	}

	return nil
}

func CreateDecoder(codecType CodecType, r io.Reader) Decoder {
	switch codecType {
	case CodecType_JSON:
		return json.NewDecoderJson(r)
	case CodecType_PROTO:
		return proto.NewDecoderProto(r)
	case CodecType_RLP:
		return rlp.NewDecoderRlp(r)
	default:
		panic(fmt.Errorf("invalid codec type %d when CreateDecoder", codecType).Error())
	}

	return nil
}
