// Code generated by protoc-gen-gogo. DO NOT EDIT.
// source: universal.proto

package universal

import (
	fmt "fmt"
	_ "github.com/gogo/protobuf/gogoproto"
	proto "github.com/golang/protobuf/proto"
	io "io"
	math "math"
	math_bits "math/bits"
)

// Reference imports to suppress errors if they are not otherwise used.
var _ = proto.Marshal
var _ = fmt.Errorf
var _ = math.Inf

// This is a compile-time assertion to ensure that this generated file
// is compatible with the proto package it is being compiled against.
// A compilation error at this line likely means your copy of the
// proto package needs to be updated.
const _ = proto.ProtoPackageIsVersion3 // please upgrade the proto package

type TransactionResultUniversal_ResultStatus int32

const (
	TransactionResultUniversal_OK  TransactionResultUniversal_ResultStatus = 0
	TransactionResultUniversal_Err TransactionResultUniversal_ResultStatus = 1
)

var TransactionResultUniversal_ResultStatus_name = map[int32]string{
	0: "OK",
	1: "Err",
}

var TransactionResultUniversal_ResultStatus_value = map[string]int32{
	"OK":  0,
	"Err": 1,
}

func (x TransactionResultUniversal_ResultStatus) String() string {
	return proto.EnumName(TransactionResultUniversal_ResultStatus_name, int32(x))
}

func (TransactionResultUniversal_ResultStatus) EnumDescriptor() ([]byte, []int) {
	return fileDescriptor_429946c34de6d866, []int{3, 0}
}

type TransactionUniversalHead struct {
	Version              uint32   `protobuf:"varint,1,opt,name=Version,proto3" json:"version"`
	FeePayer             []byte   `protobuf:"bytes,2,opt,name=FeePayer,proto3" json:"feePayer,omitempty"`
	Nonce                uint64   `protobuf:"varint,3,opt,name=Nonce,proto3" json:"nonce"`
	GasPrice             uint64   `protobuf:"varint,4,opt,name=GasPrice,proto3" json:"gasPrice,omitempty"`
	GasLimit             uint64   `protobuf:"varint,5,opt,name=GasLimit,proto3" json:"gasLimit,omitempty"`
	Type                 uint32   `protobuf:"varint,6,opt,name=Type,proto3" json:"type"`
	FeePayerSignature    []byte   `protobuf:"bytes,7,opt,name=FeePayerSignature,proto3" json:"feePayerSignature,omitempty"`
	Options              uint32   `protobuf:"varint,8,opt,name=Options,proto3" json:"options,omitempty"`
	XXX_NoUnkeyedLiteral struct{} `json:"-"`
	XXX_unrecognized     []byte   `json:"-"`
	XXX_sizecache        int32    `json:"-"`
}

func (m *TransactionUniversalHead) Reset()         { *m = TransactionUniversalHead{} }
func (m *TransactionUniversalHead) String() string { return proto.CompactTextString(m) }
func (*TransactionUniversalHead) ProtoMessage()    {}
func (*TransactionUniversalHead) Descriptor() ([]byte, []int) {
	return fileDescriptor_429946c34de6d866, []int{0}
}
func (m *TransactionUniversalHead) XXX_Unmarshal(b []byte) error {
	return m.Unmarshal(b)
}
func (m *TransactionUniversalHead) XXX_Marshal(b []byte, deterministic bool) ([]byte, error) {
	b = b[:cap(b)]
	n, err := m.MarshalToSizedBuffer(b)
	if err != nil {
		return nil, err
	}
	return b[:n], nil
}
func (m *TransactionUniversalHead) XXX_Merge(src proto.Message) {
	xxx_messageInfo_TransactionUniversalHead.Merge(m, src)
}
func (m *TransactionUniversalHead) XXX_Size() int {
	return m.Size()
}
func (m *TransactionUniversalHead) XXX_DiscardUnknown() {
	xxx_messageInfo_TransactionUniversalHead.DiscardUnknown(m)
}

var xxx_messageInfo_TransactionUniversalHead proto.InternalMessageInfo

func (m *TransactionUniversalHead) GetVersion() uint32 {
	if m != nil {
		return m.Version
	}
	return 0
}

func (m *TransactionUniversalHead) GetFeePayer() []byte {
	if m != nil {
		return m.FeePayer
	}
	return nil
}

func (m *TransactionUniversalHead) GetNonce() uint64 {
	if m != nil {
		return m.Nonce
	}
	return 0
}

func (m *TransactionUniversalHead) GetGasPrice() uint64 {
	if m != nil {
		return m.GasPrice
	}
	return 0
}

func (m *TransactionUniversalHead) GetGasLimit() uint64 {
	if m != nil {
		return m.GasLimit
	}
	return 0
}

func (m *TransactionUniversalHead) GetType() uint32 {
	if m != nil {
		return m.Type
	}
	return 0
}

func (m *TransactionUniversalHead) GetFeePayerSignature() []byte {
	if m != nil {
		return m.FeePayerSignature
	}
	return nil
}

func (m *TransactionUniversalHead) GetOptions() uint32 {
	if m != nil {
		return m.Options
	}
	return 0
}

type TransactionUniversalData struct {
	Specification        []byte   `protobuf:"bytes,1,opt,name=Specification,proto3" json:"specification"`
	XXX_NoUnkeyedLiteral struct{} `json:"-"`
	XXX_unrecognized     []byte   `json:"-"`
	XXX_sizecache        int32    `json:"-"`
}

func (m *TransactionUniversalData) Reset()         { *m = TransactionUniversalData{} }
func (m *TransactionUniversalData) String() string { return proto.CompactTextString(m) }
func (*TransactionUniversalData) ProtoMessage()    {}
func (*TransactionUniversalData) Descriptor() ([]byte, []int) {
	return fileDescriptor_429946c34de6d866, []int{1}
}
func (m *TransactionUniversalData) XXX_Unmarshal(b []byte) error {
	return m.Unmarshal(b)
}
func (m *TransactionUniversalData) XXX_Marshal(b []byte, deterministic bool) ([]byte, error) {
	b = b[:cap(b)]
	n, err := m.MarshalToSizedBuffer(b)
	if err != nil {
		return nil, err
	}
	return b[:n], nil
}
func (m *TransactionUniversalData) XXX_Merge(src proto.Message) {
	xxx_messageInfo_TransactionUniversalData.Merge(m, src)
}
func (m *TransactionUniversalData) XXX_Size() int {
	return m.Size()
}
func (m *TransactionUniversalData) XXX_DiscardUnknown() {
	xxx_messageInfo_TransactionUniversalData.DiscardUnknown(m)
}

var xxx_messageInfo_TransactionUniversalData proto.InternalMessageInfo

func (m *TransactionUniversalData) GetSpecification() []byte {
	if m != nil {
		return m.Specification
	}
	return nil
}

type TransactionUniversal struct {
	Head                 *TransactionUniversalHead `protobuf:"bytes,1,opt,name=Head,proto3" json:"head"`
	Data                 *TransactionUniversalData `protobuf:"bytes,2,opt,name=Data,proto3" json:"data"`
	XXX_NoUnkeyedLiteral struct{}                  `json:"-"`
	XXX_unrecognized     []byte                    `json:"-"`
	XXX_sizecache        int32                     `json:"-"`
}

func (m *TransactionUniversal) Reset()         { *m = TransactionUniversal{} }
func (m *TransactionUniversal) String() string { return proto.CompactTextString(m) }
func (*TransactionUniversal) ProtoMessage()    {}
func (*TransactionUniversal) Descriptor() ([]byte, []int) {
	return fileDescriptor_429946c34de6d866, []int{2}
}
func (m *TransactionUniversal) XXX_Unmarshal(b []byte) error {
	return m.Unmarshal(b)
}
func (m *TransactionUniversal) XXX_Marshal(b []byte, deterministic bool) ([]byte, error) {
	b = b[:cap(b)]
	n, err := m.MarshalToSizedBuffer(b)
	if err != nil {
		return nil, err
	}
	return b[:n], nil
}
func (m *TransactionUniversal) XXX_Merge(src proto.Message) {
	xxx_messageInfo_TransactionUniversal.Merge(m, src)
}
func (m *TransactionUniversal) XXX_Size() int {
	return m.Size()
}
func (m *TransactionUniversal) XXX_DiscardUnknown() {
	xxx_messageInfo_TransactionUniversal.DiscardUnknown(m)
}

var xxx_messageInfo_TransactionUniversal proto.InternalMessageInfo

func (m *TransactionUniversal) GetHead() *TransactionUniversalHead {
	if m != nil {
		return m.Head
	}
	return nil
}

func (m *TransactionUniversal) GetData() *TransactionUniversalData {
	if m != nil {
		return m.Data
	}
	return nil
}

type TransactionResultUniversal struct {
	Version              uint32                                  `protobuf:"varint,1,opt,name=Version,proto3" json:"version"`
	TxHash               []byte                                  `protobuf:"bytes,2,opt,name=TxHash,proto3" json:"txHash"`
	GasUsed              uint64                                  `protobuf:"varint,3,opt,name=GasUsed,proto3" json:"gasUsed"`
	ErrString            []byte                                  `protobuf:"bytes,4,opt,name=ErrString,proto3" json:"errString"`
	Status               TransactionResultUniversal_ResultStatus `protobuf:"varint,5,opt,name=Status,proto3,enum=proto.TransactionResultUniversal_ResultStatus" json:"status"`
	XXX_NoUnkeyedLiteral struct{}                                `json:"-"`
	XXX_unrecognized     []byte                                  `json:"-"`
	XXX_sizecache        int32                                   `json:"-"`
}

func (m *TransactionResultUniversal) Reset()         { *m = TransactionResultUniversal{} }
func (m *TransactionResultUniversal) String() string { return proto.CompactTextString(m) }
func (*TransactionResultUniversal) ProtoMessage()    {}
func (*TransactionResultUniversal) Descriptor() ([]byte, []int) {
	return fileDescriptor_429946c34de6d866, []int{3}
}
func (m *TransactionResultUniversal) XXX_Unmarshal(b []byte) error {
	return m.Unmarshal(b)
}
func (m *TransactionResultUniversal) XXX_Marshal(b []byte, deterministic bool) ([]byte, error) {
	b = b[:cap(b)]
	n, err := m.MarshalToSizedBuffer(b)
	if err != nil {
		return nil, err
	}
	return b[:n], nil
}
func (m *TransactionResultUniversal) XXX_Merge(src proto.Message) {
	xxx_messageInfo_TransactionResultUniversal.Merge(m, src)
}
func (m *TransactionResultUniversal) XXX_Size() int {
	return m.Size()
}
func (m *TransactionResultUniversal) XXX_DiscardUnknown() {
	xxx_messageInfo_TransactionResultUniversal.DiscardUnknown(m)
}

var xxx_messageInfo_TransactionResultUniversal proto.InternalMessageInfo

func (m *TransactionResultUniversal) GetVersion() uint32 {
	if m != nil {
		return m.Version
	}
	return 0
}

func (m *TransactionResultUniversal) GetTxHash() []byte {
	if m != nil {
		return m.TxHash
	}
	return nil
}

func (m *TransactionResultUniversal) GetGasUsed() uint64 {
	if m != nil {
		return m.GasUsed
	}
	return 0
}

func (m *TransactionResultUniversal) GetErrString() []byte {
	if m != nil {
		return m.ErrString
	}
	return nil
}

func (m *TransactionResultUniversal) GetStatus() TransactionResultUniversal_ResultStatus {
	if m != nil {
		return m.Status
	}
	return TransactionResultUniversal_OK
}

func init() {
	proto.RegisterEnum("proto.TransactionResultUniversal_ResultStatus", TransactionResultUniversal_ResultStatus_name, TransactionResultUniversal_ResultStatus_value)
	proto.RegisterType((*TransactionUniversalHead)(nil), "proto.TransactionUniversalHead")
	proto.RegisterType((*TransactionUniversalData)(nil), "proto.TransactionUniversalData")
	proto.RegisterType((*TransactionUniversal)(nil), "proto.TransactionUniversal")
	proto.RegisterType((*TransactionResultUniversal)(nil), "proto.TransactionResultUniversal")
}

func init() { proto.RegisterFile("universal.proto", fileDescriptor_429946c34de6d866) }

var fileDescriptor_429946c34de6d866 = []byte{
	// 532 bytes of a gzipped FileDescriptorProto
	0x1f, 0x8b, 0x08, 0x00, 0x00, 0x00, 0x00, 0x00, 0x02, 0xff, 0x8c, 0x53, 0xcd, 0x6e, 0xd3, 0x4c,
	0x14, 0xed, 0xe4, 0xc7, 0x49, 0x6e, 0x93, 0xef, 0x6b, 0x46, 0x80, 0xac, 0x82, 0x32, 0x91, 0x25,
	0xa4, 0x48, 0xa0, 0x20, 0x85, 0x05, 0x2b, 0x36, 0x16, 0xa5, 0x95, 0xf8, 0x69, 0x35, 0x49, 0x59,
	0xb0, 0x1b, 0x9c, 0x89, 0x3b, 0x52, 0x63, 0x5b, 0x33, 0x93, 0x8a, 0xbc, 0x07, 0x0b, 0x16, 0x3c,
	0x04, 0x8f, 0xc1, 0x92, 0x25, 0x2b, 0x0b, 0x85, 0x9d, 0x9f, 0x02, 0x79, 0xc6, 0x6e, 0xdd, 0xb4,
	0x20, 0x56, 0xb6, 0xcf, 0x3d, 0xe7, 0xcc, 0xbd, 0x73, 0x8f, 0xe1, 0xff, 0x55, 0x24, 0x2e, 0xb8,
	0x54, 0xec, 0x7c, 0x9c, 0xc8, 0x58, 0xc7, 0xb8, 0x69, 0x1e, 0xfb, 0x10, 0xc6, 0x61, 0x6c, 0x21,
	0xef, 0x4b, 0x1d, 0xdc, 0x99, 0x64, 0x91, 0x62, 0x81, 0x16, 0x71, 0x74, 0x5a, 0x2a, 0x8e, 0x38,
	0x9b, 0xe3, 0x87, 0xd0, 0x7a, 0xc7, 0xa5, 0x12, 0x71, 0xe4, 0xa2, 0x21, 0x1a, 0xf5, 0xfc, 0xdd,
	0x2c, 0x25, 0xad, 0x0b, 0x0b, 0xd1, 0xb2, 0x86, 0x27, 0xd0, 0x7e, 0xc9, 0xf9, 0x09, 0x5b, 0x73,
	0xe9, 0xd6, 0x86, 0x68, 0xd4, 0xf5, 0xef, 0x65, 0x29, 0xc1, 0x8b, 0x02, 0x7b, 0x1c, 0x2f, 0x85,
	0xe6, 0xcb, 0x44, 0xaf, 0xe9, 0x25, 0x0f, 0x13, 0x68, 0xbe, 0x8d, 0xa3, 0x80, 0xbb, 0xf5, 0x21,
	0x1a, 0x35, 0xfc, 0x4e, 0x96, 0x92, 0x66, 0x94, 0x03, 0xd4, 0xe2, 0xb9, 0xe9, 0x21, 0x53, 0x27,
	0x52, 0x04, 0xdc, 0x6d, 0x18, 0x8e, 0x31, 0x0d, 0x0b, 0xac, 0x6a, 0x5a, 0xf2, 0x0a, 0xcd, 0x6b,
	0xb1, 0x14, 0xda, 0x6d, 0x5e, 0xd3, 0x18, 0x6c, 0x4b, 0x63, 0x30, 0xfc, 0x00, 0x1a, 0xb3, 0x75,
	0xc2, 0x5d, 0xc7, 0x0c, 0xd8, 0xce, 0x52, 0xd2, 0xd0, 0xeb, 0x84, 0x53, 0x83, 0xe2, 0x37, 0xd0,
	0x2f, 0x5b, 0x9e, 0x8a, 0x30, 0x62, 0x7a, 0x25, 0xb9, 0xdb, 0x32, 0x33, 0x92, 0x2c, 0x25, 0xf7,
	0x17, 0xdb, 0xc5, 0xca, 0x19, 0x37, 0x95, 0xf8, 0x09, 0xb4, 0x8e, 0x93, 0xfc, 0x9e, 0x95, 0xdb,
	0x36, 0xe7, 0xdd, 0xcd, 0x52, 0xd2, 0x8f, 0x2d, 0x54, 0x91, 0x96, 0x2c, 0x6f, 0x7a, 0xfb, 0x76,
	0x5e, 0x30, 0xcd, 0xf0, 0x33, 0xe8, 0x4d, 0x13, 0x1e, 0x88, 0x85, 0x08, 0x98, 0x2e, 0x77, 0xd4,
	0xf5, 0xfb, 0x59, 0x4a, 0x7a, 0xaa, 0x5a, 0xa0, 0xd7, 0x79, 0xde, 0x27, 0x04, 0x77, 0x6e, 0x73,
	0xc5, 0xcf, 0xa1, 0x91, 0xef, 0xdd, 0x18, 0xed, 0x4e, 0x88, 0x8d, 0xc8, 0xf8, 0x4f, 0xf1, 0xb0,
	0x97, 0x75, 0xc6, 0xd9, 0x9c, 0x1a, 0x59, 0x2e, 0xcf, 0x1b, 0x33, 0x19, 0xf8, 0xbb, 0x3c, 0xa7,
	0x59, 0xf9, 0x9c, 0x69, 0x46, 0x8d, 0xcc, 0xfb, 0x5a, 0x83, 0xfd, 0x0a, 0x99, 0x72, 0xb5, 0x3a,
	0xd7, 0x57, 0xcd, 0xfd, 0x63, 0x18, 0x3d, 0x70, 0x66, 0x1f, 0x8f, 0x98, 0x3a, 0x2b, 0xa2, 0x08,
	0x59, 0x4a, 0x1c, 0x6d, 0x10, 0x5a, 0x54, 0x72, 0xab, 0x43, 0xa6, 0x4e, 0x15, 0x9f, 0x17, 0xf1,
	0x33, 0x56, 0xa1, 0x85, 0x68, 0x59, 0xc3, 0x8f, 0xa0, 0x73, 0x20, 0xe5, 0x54, 0x4b, 0x11, 0x85,
	0x26, 0x83, 0x5d, 0xbf, 0x97, 0xa5, 0xa4, 0xc3, 0x4b, 0x90, 0x5e, 0xd5, 0x31, 0x05, 0x67, 0xaa,
	0x99, 0x5e, 0x29, 0x93, 0xbc, 0xff, 0x26, 0xe3, 0x9b, 0xe3, 0x6f, 0x4d, 0x34, 0xb6, 0xdf, 0x56,
	0x65, 0xfb, 0x54, 0xe6, 0x9d, 0x16, 0x4e, 0x1e, 0x81, 0x6e, 0x95, 0x83, 0x1d, 0xa8, 0x1d, 0xbf,
	0xda, 0xdb, 0xc1, 0x2d, 0xa8, 0x1f, 0x48, 0xb9, 0x87, 0x7c, 0xf2, 0x6d, 0x33, 0x40, 0xdf, 0x37,
	0x03, 0xf4, 0x63, 0x33, 0x40, 0x3f, 0x37, 0x03, 0xf4, 0xf9, 0xd7, 0x60, 0xe7, 0x7d, 0xe7, 0xf2,
	0xbf, 0xff, 0xe0, 0x98, 0x26, 0x9e, 0xfe, 0x0e, 0x00, 0x00, 0xff, 0xff, 0x6b, 0x40, 0x4c, 0x0e,
	0x0b, 0x04, 0x00, 0x00,
}

func (m *TransactionUniversalHead) Marshal() (dAtA []byte, err error) {
	size := m.Size()
	dAtA = make([]byte, size)
	n, err := m.MarshalToSizedBuffer(dAtA[:size])
	if err != nil {
		return nil, err
	}
	return dAtA[:n], nil
}

func (m *TransactionUniversalHead) MarshalTo(dAtA []byte) (int, error) {
	size := m.Size()
	return m.MarshalToSizedBuffer(dAtA[:size])
}

func (m *TransactionUniversalHead) MarshalToSizedBuffer(dAtA []byte) (int, error) {
	i := len(dAtA)
	_ = i
	var l int
	_ = l
	if m.XXX_unrecognized != nil {
		i -= len(m.XXX_unrecognized)
		copy(dAtA[i:], m.XXX_unrecognized)
	}
	if m.Options != 0 {
		i = encodeVarintUniversal(dAtA, i, uint64(m.Options))
		i--
		dAtA[i] = 0x40
	}
	if len(m.FeePayerSignature) > 0 {
		i -= len(m.FeePayerSignature)
		copy(dAtA[i:], m.FeePayerSignature)
		i = encodeVarintUniversal(dAtA, i, uint64(len(m.FeePayerSignature)))
		i--
		dAtA[i] = 0x3a
	}
	if m.Type != 0 {
		i = encodeVarintUniversal(dAtA, i, uint64(m.Type))
		i--
		dAtA[i] = 0x30
	}
	if m.GasLimit != 0 {
		i = encodeVarintUniversal(dAtA, i, uint64(m.GasLimit))
		i--
		dAtA[i] = 0x28
	}
	if m.GasPrice != 0 {
		i = encodeVarintUniversal(dAtA, i, uint64(m.GasPrice))
		i--
		dAtA[i] = 0x20
	}
	if m.Nonce != 0 {
		i = encodeVarintUniversal(dAtA, i, uint64(m.Nonce))
		i--
		dAtA[i] = 0x18
	}
	if len(m.FeePayer) > 0 {
		i -= len(m.FeePayer)
		copy(dAtA[i:], m.FeePayer)
		i = encodeVarintUniversal(dAtA, i, uint64(len(m.FeePayer)))
		i--
		dAtA[i] = 0x12
	}
	if m.Version != 0 {
		i = encodeVarintUniversal(dAtA, i, uint64(m.Version))
		i--
		dAtA[i] = 0x8
	}
	return len(dAtA) - i, nil
}

func (m *TransactionUniversalData) Marshal() (dAtA []byte, err error) {
	size := m.Size()
	dAtA = make([]byte, size)
	n, err := m.MarshalToSizedBuffer(dAtA[:size])
	if err != nil {
		return nil, err
	}
	return dAtA[:n], nil
}

func (m *TransactionUniversalData) MarshalTo(dAtA []byte) (int, error) {
	size := m.Size()
	return m.MarshalToSizedBuffer(dAtA[:size])
}

func (m *TransactionUniversalData) MarshalToSizedBuffer(dAtA []byte) (int, error) {
	i := len(dAtA)
	_ = i
	var l int
	_ = l
	if m.XXX_unrecognized != nil {
		i -= len(m.XXX_unrecognized)
		copy(dAtA[i:], m.XXX_unrecognized)
	}
	if len(m.Specification) > 0 {
		i -= len(m.Specification)
		copy(dAtA[i:], m.Specification)
		i = encodeVarintUniversal(dAtA, i, uint64(len(m.Specification)))
		i--
		dAtA[i] = 0xa
	}
	return len(dAtA) - i, nil
}

func (m *TransactionUniversal) Marshal() (dAtA []byte, err error) {
	size := m.Size()
	dAtA = make([]byte, size)
	n, err := m.MarshalToSizedBuffer(dAtA[:size])
	if err != nil {
		return nil, err
	}
	return dAtA[:n], nil
}

func (m *TransactionUniversal) MarshalTo(dAtA []byte) (int, error) {
	size := m.Size()
	return m.MarshalToSizedBuffer(dAtA[:size])
}

func (m *TransactionUniversal) MarshalToSizedBuffer(dAtA []byte) (int, error) {
	i := len(dAtA)
	_ = i
	var l int
	_ = l
	if m.XXX_unrecognized != nil {
		i -= len(m.XXX_unrecognized)
		copy(dAtA[i:], m.XXX_unrecognized)
	}
	if m.Data != nil {
		{
			size, err := m.Data.MarshalToSizedBuffer(dAtA[:i])
			if err != nil {
				return 0, err
			}
			i -= size
			i = encodeVarintUniversal(dAtA, i, uint64(size))
		}
		i--
		dAtA[i] = 0x12
	}
	if m.Head != nil {
		{
			size, err := m.Head.MarshalToSizedBuffer(dAtA[:i])
			if err != nil {
				return 0, err
			}
			i -= size
			i = encodeVarintUniversal(dAtA, i, uint64(size))
		}
		i--
		dAtA[i] = 0xa
	}
	return len(dAtA) - i, nil
}

func (m *TransactionResultUniversal) Marshal() (dAtA []byte, err error) {
	size := m.Size()
	dAtA = make([]byte, size)
	n, err := m.MarshalToSizedBuffer(dAtA[:size])
	if err != nil {
		return nil, err
	}
	return dAtA[:n], nil
}

func (m *TransactionResultUniversal) MarshalTo(dAtA []byte) (int, error) {
	size := m.Size()
	return m.MarshalToSizedBuffer(dAtA[:size])
}

func (m *TransactionResultUniversal) MarshalToSizedBuffer(dAtA []byte) (int, error) {
	i := len(dAtA)
	_ = i
	var l int
	_ = l
	if m.XXX_unrecognized != nil {
		i -= len(m.XXX_unrecognized)
		copy(dAtA[i:], m.XXX_unrecognized)
	}
	if m.Status != 0 {
		i = encodeVarintUniversal(dAtA, i, uint64(m.Status))
		i--
		dAtA[i] = 0x28
	}
	if len(m.ErrString) > 0 {
		i -= len(m.ErrString)
		copy(dAtA[i:], m.ErrString)
		i = encodeVarintUniversal(dAtA, i, uint64(len(m.ErrString)))
		i--
		dAtA[i] = 0x22
	}
	if m.GasUsed != 0 {
		i = encodeVarintUniversal(dAtA, i, uint64(m.GasUsed))
		i--
		dAtA[i] = 0x18
	}
	if len(m.TxHash) > 0 {
		i -= len(m.TxHash)
		copy(dAtA[i:], m.TxHash)
		i = encodeVarintUniversal(dAtA, i, uint64(len(m.TxHash)))
		i--
		dAtA[i] = 0x12
	}
	if m.Version != 0 {
		i = encodeVarintUniversal(dAtA, i, uint64(m.Version))
		i--
		dAtA[i] = 0x8
	}
	return len(dAtA) - i, nil
}

func encodeVarintUniversal(dAtA []byte, offset int, v uint64) int {
	offset -= sovUniversal(v)
	base := offset
	for v >= 1<<7 {
		dAtA[offset] = uint8(v&0x7f | 0x80)
		v >>= 7
		offset++
	}
	dAtA[offset] = uint8(v)
	return base
}
func (m *TransactionUniversalHead) Size() (n int) {
	if m == nil {
		return 0
	}
	var l int
	_ = l
	if m.Version != 0 {
		n += 1 + sovUniversal(uint64(m.Version))
	}
	l = len(m.FeePayer)
	if l > 0 {
		n += 1 + l + sovUniversal(uint64(l))
	}
	if m.Nonce != 0 {
		n += 1 + sovUniversal(uint64(m.Nonce))
	}
	if m.GasPrice != 0 {
		n += 1 + sovUniversal(uint64(m.GasPrice))
	}
	if m.GasLimit != 0 {
		n += 1 + sovUniversal(uint64(m.GasLimit))
	}
	if m.Type != 0 {
		n += 1 + sovUniversal(uint64(m.Type))
	}
	l = len(m.FeePayerSignature)
	if l > 0 {
		n += 1 + l + sovUniversal(uint64(l))
	}
	if m.Options != 0 {
		n += 1 + sovUniversal(uint64(m.Options))
	}
	if m.XXX_unrecognized != nil {
		n += len(m.XXX_unrecognized)
	}
	return n
}

func (m *TransactionUniversalData) Size() (n int) {
	if m == nil {
		return 0
	}
	var l int
	_ = l
	l = len(m.Specification)
	if l > 0 {
		n += 1 + l + sovUniversal(uint64(l))
	}
	if m.XXX_unrecognized != nil {
		n += len(m.XXX_unrecognized)
	}
	return n
}

func (m *TransactionUniversal) Size() (n int) {
	if m == nil {
		return 0
	}
	var l int
	_ = l
	if m.Head != nil {
		l = m.Head.Size()
		n += 1 + l + sovUniversal(uint64(l))
	}
	if m.Data != nil {
		l = m.Data.Size()
		n += 1 + l + sovUniversal(uint64(l))
	}
	if m.XXX_unrecognized != nil {
		n += len(m.XXX_unrecognized)
	}
	return n
}

func (m *TransactionResultUniversal) Size() (n int) {
	if m == nil {
		return 0
	}
	var l int
	_ = l
	if m.Version != 0 {
		n += 1 + sovUniversal(uint64(m.Version))
	}
	l = len(m.TxHash)
	if l > 0 {
		n += 1 + l + sovUniversal(uint64(l))
	}
	if m.GasUsed != 0 {
		n += 1 + sovUniversal(uint64(m.GasUsed))
	}
	l = len(m.ErrString)
	if l > 0 {
		n += 1 + l + sovUniversal(uint64(l))
	}
	if m.Status != 0 {
		n += 1 + sovUniversal(uint64(m.Status))
	}
	if m.XXX_unrecognized != nil {
		n += len(m.XXX_unrecognized)
	}
	return n
}

func sovUniversal(x uint64) (n int) {
	return (math_bits.Len64(x|1) + 6) / 7
}
func sozUniversal(x uint64) (n int) {
	return sovUniversal(uint64((x << 1) ^ uint64((int64(x) >> 63))))
}
func (m *TransactionUniversalHead) Unmarshal(dAtA []byte) error {
	l := len(dAtA)
	iNdEx := 0
	for iNdEx < l {
		preIndex := iNdEx
		var wire uint64
		for shift := uint(0); ; shift += 7 {
			if shift >= 64 {
				return ErrIntOverflowUniversal
			}
			if iNdEx >= l {
				return io.ErrUnexpectedEOF
			}
			b := dAtA[iNdEx]
			iNdEx++
			wire |= uint64(b&0x7F) << shift
			if b < 0x80 {
				break
			}
		}
		fieldNum := int32(wire >> 3)
		wireType := int(wire & 0x7)
		if wireType == 4 {
			return fmt.Errorf("proto: TransactionUniversalHead: wiretype end group for non-group")
		}
		if fieldNum <= 0 {
			return fmt.Errorf("proto: TransactionUniversalHead: illegal tag %d (wire type %d)", fieldNum, wire)
		}
		switch fieldNum {
		case 1:
			if wireType != 0 {
				return fmt.Errorf("proto: wrong wireType = %d for field Version", wireType)
			}
			m.Version = 0
			for shift := uint(0); ; shift += 7 {
				if shift >= 64 {
					return ErrIntOverflowUniversal
				}
				if iNdEx >= l {
					return io.ErrUnexpectedEOF
				}
				b := dAtA[iNdEx]
				iNdEx++
				m.Version |= uint32(b&0x7F) << shift
				if b < 0x80 {
					break
				}
			}
		case 2:
			if wireType != 2 {
				return fmt.Errorf("proto: wrong wireType = %d for field FeePayer", wireType)
			}
			var byteLen int
			for shift := uint(0); ; shift += 7 {
				if shift >= 64 {
					return ErrIntOverflowUniversal
				}
				if iNdEx >= l {
					return io.ErrUnexpectedEOF
				}
				b := dAtA[iNdEx]
				iNdEx++
				byteLen |= int(b&0x7F) << shift
				if b < 0x80 {
					break
				}
			}
			if byteLen < 0 {
				return ErrInvalidLengthUniversal
			}
			postIndex := iNdEx + byteLen
			if postIndex < 0 {
				return ErrInvalidLengthUniversal
			}
			if postIndex > l {
				return io.ErrUnexpectedEOF
			}
			m.FeePayer = append(m.FeePayer[:0], dAtA[iNdEx:postIndex]...)
			if m.FeePayer == nil {
				m.FeePayer = []byte{}
			}
			iNdEx = postIndex
		case 3:
			if wireType != 0 {
				return fmt.Errorf("proto: wrong wireType = %d for field Nonce", wireType)
			}
			m.Nonce = 0
			for shift := uint(0); ; shift += 7 {
				if shift >= 64 {
					return ErrIntOverflowUniversal
				}
				if iNdEx >= l {
					return io.ErrUnexpectedEOF
				}
				b := dAtA[iNdEx]
				iNdEx++
				m.Nonce |= uint64(b&0x7F) << shift
				if b < 0x80 {
					break
				}
			}
		case 4:
			if wireType != 0 {
				return fmt.Errorf("proto: wrong wireType = %d for field GasPrice", wireType)
			}
			m.GasPrice = 0
			for shift := uint(0); ; shift += 7 {
				if shift >= 64 {
					return ErrIntOverflowUniversal
				}
				if iNdEx >= l {
					return io.ErrUnexpectedEOF
				}
				b := dAtA[iNdEx]
				iNdEx++
				m.GasPrice |= uint64(b&0x7F) << shift
				if b < 0x80 {
					break
				}
			}
		case 5:
			if wireType != 0 {
				return fmt.Errorf("proto: wrong wireType = %d for field GasLimit", wireType)
			}
			m.GasLimit = 0
			for shift := uint(0); ; shift += 7 {
				if shift >= 64 {
					return ErrIntOverflowUniversal
				}
				if iNdEx >= l {
					return io.ErrUnexpectedEOF
				}
				b := dAtA[iNdEx]
				iNdEx++
				m.GasLimit |= uint64(b&0x7F) << shift
				if b < 0x80 {
					break
				}
			}
		case 6:
			if wireType != 0 {
				return fmt.Errorf("proto: wrong wireType = %d for field Type", wireType)
			}
			m.Type = 0
			for shift := uint(0); ; shift += 7 {
				if shift >= 64 {
					return ErrIntOverflowUniversal
				}
				if iNdEx >= l {
					return io.ErrUnexpectedEOF
				}
				b := dAtA[iNdEx]
				iNdEx++
				m.Type |= uint32(b&0x7F) << shift
				if b < 0x80 {
					break
				}
			}
		case 7:
			if wireType != 2 {
				return fmt.Errorf("proto: wrong wireType = %d for field FeePayerSignature", wireType)
			}
			var byteLen int
			for shift := uint(0); ; shift += 7 {
				if shift >= 64 {
					return ErrIntOverflowUniversal
				}
				if iNdEx >= l {
					return io.ErrUnexpectedEOF
				}
				b := dAtA[iNdEx]
				iNdEx++
				byteLen |= int(b&0x7F) << shift
				if b < 0x80 {
					break
				}
			}
			if byteLen < 0 {
				return ErrInvalidLengthUniversal
			}
			postIndex := iNdEx + byteLen
			if postIndex < 0 {
				return ErrInvalidLengthUniversal
			}
			if postIndex > l {
				return io.ErrUnexpectedEOF
			}
			m.FeePayerSignature = append(m.FeePayerSignature[:0], dAtA[iNdEx:postIndex]...)
			if m.FeePayerSignature == nil {
				m.FeePayerSignature = []byte{}
			}
			iNdEx = postIndex
		case 8:
			if wireType != 0 {
				return fmt.Errorf("proto: wrong wireType = %d for field Options", wireType)
			}
			m.Options = 0
			for shift := uint(0); ; shift += 7 {
				if shift >= 64 {
					return ErrIntOverflowUniversal
				}
				if iNdEx >= l {
					return io.ErrUnexpectedEOF
				}
				b := dAtA[iNdEx]
				iNdEx++
				m.Options |= uint32(b&0x7F) << shift
				if b < 0x80 {
					break
				}
			}
		default:
			iNdEx = preIndex
			skippy, err := skipUniversal(dAtA[iNdEx:])
			if err != nil {
				return err
			}
			if (skippy < 0) || (iNdEx+skippy) < 0 {
				return ErrInvalidLengthUniversal
			}
			if (iNdEx + skippy) > l {
				return io.ErrUnexpectedEOF
			}
			m.XXX_unrecognized = append(m.XXX_unrecognized, dAtA[iNdEx:iNdEx+skippy]...)
			iNdEx += skippy
		}
	}

	if iNdEx > l {
		return io.ErrUnexpectedEOF
	}
	return nil
}
func (m *TransactionUniversalData) Unmarshal(dAtA []byte) error {
	l := len(dAtA)
	iNdEx := 0
	for iNdEx < l {
		preIndex := iNdEx
		var wire uint64
		for shift := uint(0); ; shift += 7 {
			if shift >= 64 {
				return ErrIntOverflowUniversal
			}
			if iNdEx >= l {
				return io.ErrUnexpectedEOF
			}
			b := dAtA[iNdEx]
			iNdEx++
			wire |= uint64(b&0x7F) << shift
			if b < 0x80 {
				break
			}
		}
		fieldNum := int32(wire >> 3)
		wireType := int(wire & 0x7)
		if wireType == 4 {
			return fmt.Errorf("proto: TransactionUniversalData: wiretype end group for non-group")
		}
		if fieldNum <= 0 {
			return fmt.Errorf("proto: TransactionUniversalData: illegal tag %d (wire type %d)", fieldNum, wire)
		}
		switch fieldNum {
		case 1:
			if wireType != 2 {
				return fmt.Errorf("proto: wrong wireType = %d for field Specification", wireType)
			}
			var byteLen int
			for shift := uint(0); ; shift += 7 {
				if shift >= 64 {
					return ErrIntOverflowUniversal
				}
				if iNdEx >= l {
					return io.ErrUnexpectedEOF
				}
				b := dAtA[iNdEx]
				iNdEx++
				byteLen |= int(b&0x7F) << shift
				if b < 0x80 {
					break
				}
			}
			if byteLen < 0 {
				return ErrInvalidLengthUniversal
			}
			postIndex := iNdEx + byteLen
			if postIndex < 0 {
				return ErrInvalidLengthUniversal
			}
			if postIndex > l {
				return io.ErrUnexpectedEOF
			}
			m.Specification = append(m.Specification[:0], dAtA[iNdEx:postIndex]...)
			if m.Specification == nil {
				m.Specification = []byte{}
			}
			iNdEx = postIndex
		default:
			iNdEx = preIndex
			skippy, err := skipUniversal(dAtA[iNdEx:])
			if err != nil {
				return err
			}
			if (skippy < 0) || (iNdEx+skippy) < 0 {
				return ErrInvalidLengthUniversal
			}
			if (iNdEx + skippy) > l {
				return io.ErrUnexpectedEOF
			}
			m.XXX_unrecognized = append(m.XXX_unrecognized, dAtA[iNdEx:iNdEx+skippy]...)
			iNdEx += skippy
		}
	}

	if iNdEx > l {
		return io.ErrUnexpectedEOF
	}
	return nil
}
func (m *TransactionUniversal) Unmarshal(dAtA []byte) error {
	l := len(dAtA)
	iNdEx := 0
	for iNdEx < l {
		preIndex := iNdEx
		var wire uint64
		for shift := uint(0); ; shift += 7 {
			if shift >= 64 {
				return ErrIntOverflowUniversal
			}
			if iNdEx >= l {
				return io.ErrUnexpectedEOF
			}
			b := dAtA[iNdEx]
			iNdEx++
			wire |= uint64(b&0x7F) << shift
			if b < 0x80 {
				break
			}
		}
		fieldNum := int32(wire >> 3)
		wireType := int(wire & 0x7)
		if wireType == 4 {
			return fmt.Errorf("proto: TransactionUniversal: wiretype end group for non-group")
		}
		if fieldNum <= 0 {
			return fmt.Errorf("proto: TransactionUniversal: illegal tag %d (wire type %d)", fieldNum, wire)
		}
		switch fieldNum {
		case 1:
			if wireType != 2 {
				return fmt.Errorf("proto: wrong wireType = %d for field Head", wireType)
			}
			var msglen int
			for shift := uint(0); ; shift += 7 {
				if shift >= 64 {
					return ErrIntOverflowUniversal
				}
				if iNdEx >= l {
					return io.ErrUnexpectedEOF
				}
				b := dAtA[iNdEx]
				iNdEx++
				msglen |= int(b&0x7F) << shift
				if b < 0x80 {
					break
				}
			}
			if msglen < 0 {
				return ErrInvalidLengthUniversal
			}
			postIndex := iNdEx + msglen
			if postIndex < 0 {
				return ErrInvalidLengthUniversal
			}
			if postIndex > l {
				return io.ErrUnexpectedEOF
			}
			if m.Head == nil {
				m.Head = &TransactionUniversalHead{}
			}
			if err := m.Head.Unmarshal(dAtA[iNdEx:postIndex]); err != nil {
				return err
			}
			iNdEx = postIndex
		case 2:
			if wireType != 2 {
				return fmt.Errorf("proto: wrong wireType = %d for field Data", wireType)
			}
			var msglen int
			for shift := uint(0); ; shift += 7 {
				if shift >= 64 {
					return ErrIntOverflowUniversal
				}
				if iNdEx >= l {
					return io.ErrUnexpectedEOF
				}
				b := dAtA[iNdEx]
				iNdEx++
				msglen |= int(b&0x7F) << shift
				if b < 0x80 {
					break
				}
			}
			if msglen < 0 {
				return ErrInvalidLengthUniversal
			}
			postIndex := iNdEx + msglen
			if postIndex < 0 {
				return ErrInvalidLengthUniversal
			}
			if postIndex > l {
				return io.ErrUnexpectedEOF
			}
			if m.Data == nil {
				m.Data = &TransactionUniversalData{}
			}
			if err := m.Data.Unmarshal(dAtA[iNdEx:postIndex]); err != nil {
				return err
			}
			iNdEx = postIndex
		default:
			iNdEx = preIndex
			skippy, err := skipUniversal(dAtA[iNdEx:])
			if err != nil {
				return err
			}
			if (skippy < 0) || (iNdEx+skippy) < 0 {
				return ErrInvalidLengthUniversal
			}
			if (iNdEx + skippy) > l {
				return io.ErrUnexpectedEOF
			}
			m.XXX_unrecognized = append(m.XXX_unrecognized, dAtA[iNdEx:iNdEx+skippy]...)
			iNdEx += skippy
		}
	}

	if iNdEx > l {
		return io.ErrUnexpectedEOF
	}
	return nil
}
func (m *TransactionResultUniversal) Unmarshal(dAtA []byte) error {
	l := len(dAtA)
	iNdEx := 0
	for iNdEx < l {
		preIndex := iNdEx
		var wire uint64
		for shift := uint(0); ; shift += 7 {
			if shift >= 64 {
				return ErrIntOverflowUniversal
			}
			if iNdEx >= l {
				return io.ErrUnexpectedEOF
			}
			b := dAtA[iNdEx]
			iNdEx++
			wire |= uint64(b&0x7F) << shift
			if b < 0x80 {
				break
			}
		}
		fieldNum := int32(wire >> 3)
		wireType := int(wire & 0x7)
		if wireType == 4 {
			return fmt.Errorf("proto: TransactionResultUniversal: wiretype end group for non-group")
		}
		if fieldNum <= 0 {
			return fmt.Errorf("proto: TransactionResultUniversal: illegal tag %d (wire type %d)", fieldNum, wire)
		}
		switch fieldNum {
		case 1:
			if wireType != 0 {
				return fmt.Errorf("proto: wrong wireType = %d for field Version", wireType)
			}
			m.Version = 0
			for shift := uint(0); ; shift += 7 {
				if shift >= 64 {
					return ErrIntOverflowUniversal
				}
				if iNdEx >= l {
					return io.ErrUnexpectedEOF
				}
				b := dAtA[iNdEx]
				iNdEx++
				m.Version |= uint32(b&0x7F) << shift
				if b < 0x80 {
					break
				}
			}
		case 2:
			if wireType != 2 {
				return fmt.Errorf("proto: wrong wireType = %d for field TxHash", wireType)
			}
			var byteLen int
			for shift := uint(0); ; shift += 7 {
				if shift >= 64 {
					return ErrIntOverflowUniversal
				}
				if iNdEx >= l {
					return io.ErrUnexpectedEOF
				}
				b := dAtA[iNdEx]
				iNdEx++
				byteLen |= int(b&0x7F) << shift
				if b < 0x80 {
					break
				}
			}
			if byteLen < 0 {
				return ErrInvalidLengthUniversal
			}
			postIndex := iNdEx + byteLen
			if postIndex < 0 {
				return ErrInvalidLengthUniversal
			}
			if postIndex > l {
				return io.ErrUnexpectedEOF
			}
			m.TxHash = append(m.TxHash[:0], dAtA[iNdEx:postIndex]...)
			if m.TxHash == nil {
				m.TxHash = []byte{}
			}
			iNdEx = postIndex
		case 3:
			if wireType != 0 {
				return fmt.Errorf("proto: wrong wireType = %d for field GasUsed", wireType)
			}
			m.GasUsed = 0
			for shift := uint(0); ; shift += 7 {
				if shift >= 64 {
					return ErrIntOverflowUniversal
				}
				if iNdEx >= l {
					return io.ErrUnexpectedEOF
				}
				b := dAtA[iNdEx]
				iNdEx++
				m.GasUsed |= uint64(b&0x7F) << shift
				if b < 0x80 {
					break
				}
			}
		case 4:
			if wireType != 2 {
				return fmt.Errorf("proto: wrong wireType = %d for field ErrString", wireType)
			}
			var byteLen int
			for shift := uint(0); ; shift += 7 {
				if shift >= 64 {
					return ErrIntOverflowUniversal
				}
				if iNdEx >= l {
					return io.ErrUnexpectedEOF
				}
				b := dAtA[iNdEx]
				iNdEx++
				byteLen |= int(b&0x7F) << shift
				if b < 0x80 {
					break
				}
			}
			if byteLen < 0 {
				return ErrInvalidLengthUniversal
			}
			postIndex := iNdEx + byteLen
			if postIndex < 0 {
				return ErrInvalidLengthUniversal
			}
			if postIndex > l {
				return io.ErrUnexpectedEOF
			}
			m.ErrString = append(m.ErrString[:0], dAtA[iNdEx:postIndex]...)
			if m.ErrString == nil {
				m.ErrString = []byte{}
			}
			iNdEx = postIndex
		case 5:
			if wireType != 0 {
				return fmt.Errorf("proto: wrong wireType = %d for field Status", wireType)
			}
			m.Status = 0
			for shift := uint(0); ; shift += 7 {
				if shift >= 64 {
					return ErrIntOverflowUniversal
				}
				if iNdEx >= l {
					return io.ErrUnexpectedEOF
				}
				b := dAtA[iNdEx]
				iNdEx++
				m.Status |= TransactionResultUniversal_ResultStatus(b&0x7F) << shift
				if b < 0x80 {
					break
				}
			}
		default:
			iNdEx = preIndex
			skippy, err := skipUniversal(dAtA[iNdEx:])
			if err != nil {
				return err
			}
			if (skippy < 0) || (iNdEx+skippy) < 0 {
				return ErrInvalidLengthUniversal
			}
			if (iNdEx + skippy) > l {
				return io.ErrUnexpectedEOF
			}
			m.XXX_unrecognized = append(m.XXX_unrecognized, dAtA[iNdEx:iNdEx+skippy]...)
			iNdEx += skippy
		}
	}

	if iNdEx > l {
		return io.ErrUnexpectedEOF
	}
	return nil
}
func skipUniversal(dAtA []byte) (n int, err error) {
	l := len(dAtA)
	iNdEx := 0
	depth := 0
	for iNdEx < l {
		var wire uint64
		for shift := uint(0); ; shift += 7 {
			if shift >= 64 {
				return 0, ErrIntOverflowUniversal
			}
			if iNdEx >= l {
				return 0, io.ErrUnexpectedEOF
			}
			b := dAtA[iNdEx]
			iNdEx++
			wire |= (uint64(b) & 0x7F) << shift
			if b < 0x80 {
				break
			}
		}
		wireType := int(wire & 0x7)
		switch wireType {
		case 0:
			for shift := uint(0); ; shift += 7 {
				if shift >= 64 {
					return 0, ErrIntOverflowUniversal
				}
				if iNdEx >= l {
					return 0, io.ErrUnexpectedEOF
				}
				iNdEx++
				if dAtA[iNdEx-1] < 0x80 {
					break
				}
			}
		case 1:
			iNdEx += 8
		case 2:
			var length int
			for shift := uint(0); ; shift += 7 {
				if shift >= 64 {
					return 0, ErrIntOverflowUniversal
				}
				if iNdEx >= l {
					return 0, io.ErrUnexpectedEOF
				}
				b := dAtA[iNdEx]
				iNdEx++
				length |= (int(b) & 0x7F) << shift
				if b < 0x80 {
					break
				}
			}
			if length < 0 {
				return 0, ErrInvalidLengthUniversal
			}
			iNdEx += length
		case 3:
			depth++
		case 4:
			if depth == 0 {
				return 0, ErrUnexpectedEndOfGroupUniversal
			}
			depth--
		case 5:
			iNdEx += 4
		default:
			return 0, fmt.Errorf("proto: illegal wireType %d", wireType)
		}
		if iNdEx < 0 {
			return 0, ErrInvalidLengthUniversal
		}
		if depth == 0 {
			return iNdEx, nil
		}
	}
	return 0, io.ErrUnexpectedEOF
}

var (
	ErrInvalidLengthUniversal        = fmt.Errorf("proto: negative length found during unmarshaling")
	ErrIntOverflowUniversal          = fmt.Errorf("proto: integer overflow")
	ErrUnexpectedEndOfGroupUniversal = fmt.Errorf("proto: unexpected end of group")
)
