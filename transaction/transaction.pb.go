// Code generated by protoc-gen-gogo. DO NOT EDIT.
// source: transaction.proto

package transaction

import (
	fmt "fmt"
	proto "github.com/golang/protobuf/proto"
	io "io"
	math "math"
	math_bits "math/bits"
	"time"
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

type TransactionResult_ResultStatus int32

const (
	TransactionResult_OK  TransactionResult_ResultStatus = 0
	TransactionResult_Err TransactionResult_ResultStatus = 1
)

var TransactionResult_ResultStatus_name = map[int32]string{
	0: "OK",
	1: "Err",
}

var TransactionResult_ResultStatus_value = map[string]int32{
	"OK":  0,
	"Err": 1,
}

func (x TransactionResult_ResultStatus) String() string {
	return proto.EnumName(TransactionResult_ResultStatus_name, int32(x))
}

func (TransactionResult_ResultStatus) EnumDescriptor() ([]byte, []int) {
	return fileDescriptor_2cc4e03d2c28c490, []int{1, 0}
}

type Transaction struct {
	FromAddr             []byte    `protobuf:"bytes,1,opt,name=FromAddr,proto3" json:"from"`
	TargetAddr           []byte    `protobuf:"bytes,2,opt,name=TargetAddr,proto3" json:"target"`
	Version              uint32    `protobuf:"varint,3,opt,name=Version,proto3" json:"version"`
	ChainID              []byte    `protobuf:"bytes,4,opt,name=ChainID,proto3" json:"chainID"`
	Nonce                uint64    `protobuf:"varint,5,opt,name=Nonce,proto3" json:"nonce"`
	Value                []byte    `protobuf:"bytes,6,opt,name=Value,proto3" json:"value"`
	GasPrice             uint64    `protobuf:"varint,7,opt,name=GasPrice,proto3" json:"gasPrice,omitempty"`
	GasLimit             uint64    `protobuf:"varint,8,opt,name=GasLimit,proto3" json:"gasLimit,omitempty"`
	Data                 []byte    `protobuf:"bytes,9,opt,name=Data,proto3" json:"data,omitempty"`
	Signature            []byte    `protobuf:"bytes,10,opt,name=Signature,proto3" json:"signature,omitempty"`
	Options              uint32    `protobuf:"varint,11,opt,name=Options,proto3" json:"options,omitempty"`
	Time                 time.Time // Time first seen locally (spam avoidance)
	XXX_NoUnkeyedLiteral struct{}  `json:"-"`
	XXX_unrecognized     []byte    `json:"-"`
	XXX_sizecache        int32     `json:"-"`
}

func (m *Transaction) setTime() {
	m.Time = time.Now()
}
func (m *Transaction) Reset()         { *m = Transaction{} }
func (m *Transaction) String() string { return proto.CompactTextString(m) }
func (*Transaction) ProtoMessage()    {}
func (*Transaction) Descriptor() ([]byte, []int) {
	return fileDescriptor_2cc4e03d2c28c490, []int{0}
}
func (m *Transaction) XXX_Unmarshal(b []byte) error {
	return m.Unmarshal(b)
}

func (m *Transaction) XXX_Marshal(b []byte, deterministic bool) ([]byte, error) {
	b = b[:cap(b)]
	n, err := m.MarshalToSizedBuffer(b)
	if err != nil {
		return nil, err
	}
	return b[:n], nil
}

func (m *Transaction) XXX_Merge(src proto.Message) {
	xxx_messageInfo_Transaction.Merge(m, src)
}

func (m *Transaction) XXX_Size() int {
	return m.Size()
}

func (m *Transaction) XXX_DiscardUnknown() {
	xxx_messageInfo_Transaction.DiscardUnknown(m)
}

var xxx_messageInfo_Transaction proto.InternalMessageInfo

func (m *Transaction) GetFromAddr() []byte {
	if m != nil {
		return m.FromAddr
	}
	return nil
}

func (m *Transaction) GetTargetAddr() []byte {
	if m != nil {
		return m.TargetAddr
	}
	return nil
}

func (m *Transaction) GetVersion() uint32 {
	if m != nil {
		return m.Version
	}
	return 0
}

func (m *Transaction) GetChainID() []byte {
	if m != nil {
		return m.ChainID
	}
	return nil
}

func (m *Transaction) GetNonce() uint64 {
	if m != nil {
		return m.Nonce
	}
	return 0
}

func (m *Transaction) GetValue() []byte {
	if m != nil {
		return m.Value
	}
	return nil
}

func (m *Transaction) GetGasPrice() uint64 {
	if m != nil {
		return m.GasPrice
	}
	return 0
}

func (m *Transaction) GetGasLimit() uint64 {
	if m != nil {
		return m.GasLimit
	}
	return 0
}

func (m *Transaction) GetData() []byte {
	if m != nil {
		return m.Data
	}
	return nil
}

func (m *Transaction) GetSignature() []byte {
	if m != nil {
		return m.Signature
	}
	return nil
}

func (m *Transaction) GetOptions() uint32 {
	if m != nil {
		return m.Options
	}
	return 0
}

// TxDifference returns a new set which is the difference between a and b.
func TxDifference(a, b []*Transaction) []*Transaction {
	keep := make([]*Transaction, 0, len(a))
	remove := make(map[string]struct{})
	for _, tx := range b {
		if txId, err := tx.TxID(); err != nil {
			remove[txId] = struct{}{}
		}
	}
	for _, tx := range a {
		if txId, err := tx.TxID(); err != nil {
			if _, ok := remove[txId]; !ok {
				keep = append(keep, tx)
			}
		}
	}
	return keep
}

type TransactionResult struct {
	TxHash               []byte                         `protobuf:"bytes,1,opt,name=TxHash,proto3" json:"txHash"`
	GasUsed              uint64                         `protobuf:"varint,2,opt,name=GasUsed,proto3" json:"gasUsed"`
	Status               TransactionResult_ResultStatus `protobuf:"varint,3,opt,name=Status,proto3,enum=proto.TransactionResult_ResultStatus" json:"status"`
	XXX_NoUnkeyedLiteral struct{}                       `json:"-"`
	XXX_unrecognized     []byte                         `json:"-"`
	XXX_sizecache        int32                          `json:"-"`
}

func (m *TransactionResult) Reset()         { *m = TransactionResult{} }
func (m *TransactionResult) String() string { return proto.CompactTextString(m) }
func (*TransactionResult) ProtoMessage()    {}
func (*TransactionResult) Descriptor() ([]byte, []int) {
	return fileDescriptor_2cc4e03d2c28c490, []int{1}
}
func (m *TransactionResult) XXX_Unmarshal(b []byte) error {
	return m.Unmarshal(b)
}
func (m *TransactionResult) XXX_Marshal(b []byte, deterministic bool) ([]byte, error) {
	b = b[:cap(b)]
	n, err := m.MarshalToSizedBuffer(b)
	if err != nil {
		return nil, err
	}
	return b[:n], nil
}
func (m *TransactionResult) XXX_Merge(src proto.Message) {
	xxx_messageInfo_TransactionResult.Merge(m, src)
}
func (m *TransactionResult) XXX_Size() int {
	return m.Size()
}
func (m *TransactionResult) XXX_DiscardUnknown() {
	xxx_messageInfo_TransactionResult.DiscardUnknown(m)
}

var xxx_messageInfo_TransactionResult proto.InternalMessageInfo

func (m *TransactionResult) GetTxHash() []byte {
	if m != nil {
		return m.TxHash
	}
	return nil
}

func (m *TransactionResult) GetGasUsed() uint64 {
	if m != nil {
		return m.GasUsed
	}
	return 0
}

func (m *TransactionResult) GetStatus() TransactionResult_ResultStatus {
	if m != nil {
		return m.Status
	}
	return TransactionResult_OK
}

func init() {
	proto.RegisterEnum("proto.TransactionResult_ResultStatus", TransactionResult_ResultStatus_name, TransactionResult_ResultStatus_value)
	proto.RegisterType((*Transaction)(nil), "proto.Transaction")
	proto.RegisterType((*TransactionResult)(nil), "proto.TransactionResult")
}

func init() { proto.RegisterFile("transaction.proto", fileDescriptor_2cc4e03d2c28c490) }

var fileDescriptor_2cc4e03d2c28c490 = []byte{
	// 462 bytes of a gzipped FileDescriptorProto
	0x1f, 0x8b, 0x08, 0x00, 0x00, 0x00, 0x00, 0x00, 0x02, 0xff, 0x64, 0x92, 0x41, 0x6f, 0xd3, 0x3c,
	0x18, 0xc7, 0xe7, 0x35, 0x4d, 0xda, 0xa7, 0x7b, 0xa7, 0xd5, 0xaf, 0x00, 0x8b, 0x43, 0x5c, 0x2a,
	0x86, 0x2a, 0x84, 0x8a, 0x34, 0xc4, 0x07, 0xa0, 0x0c, 0xc6, 0x04, 0x62, 0xc8, 0x2b, 0x3b, 0x70,
	0x33, 0xad, 0xc9, 0x22, 0x2d, 0x71, 0x65, 0xbb, 0x13, 0x7c, 0x13, 0x3e, 0x12, 0xe2, 0xc4, 0x91,
	0x53, 0x84, 0xca, 0x01, 0x29, 0x9f, 0x02, 0xe5, 0x71, 0xb2, 0x05, 0x38, 0xb9, 0xfa, 0xfd, 0x7f,
	0xcf, 0xe3, 0x2a, 0x7f, 0xc3, 0xd0, 0x19, 0x99, 0x5b, 0xb9, 0x70, 0xa9, 0xce, 0xa7, 0x2b, 0xa3,
	0x9d, 0xa6, 0x5d, 0x3c, 0x6e, 0x43, 0xa2, 0x13, 0xed, 0xd1, 0xf8, 0x57, 0x07, 0x06, 0xf3, 0x6b,
	0x91, 0xde, 0x85, 0xde, 0x73, 0xa3, 0xb3, 0x27, 0xcb, 0xa5, 0x61, 0x64, 0x44, 0x26, 0x3b, 0xb3,
	0x5e, 0x59, 0xf0, 0xe0, 0x83, 0xd1, 0x99, 0xb8, 0x4a, 0xe8, 0x7d, 0x80, 0xb9, 0x34, 0x89, 0x72,
	0xe8, 0x6d, 0xa3, 0x07, 0x65, 0xc1, 0x43, 0x87, 0x54, 0xb4, 0x52, 0xba, 0x0f, 0xd1, 0x99, 0x32,
	0x36, 0xd5, 0x39, 0xeb, 0x8c, 0xc8, 0xe4, 0xbf, 0xd9, 0xa0, 0x2c, 0x78, 0x74, 0xe9, 0x91, 0x68,
	0xb2, 0x4a, 0x7b, 0x7a, 0x2e, 0xd3, 0xfc, 0xf8, 0x90, 0x05, 0xb8, 0x0f, 0xb5, 0x85, 0x47, 0xa2,
	0xc9, 0x28, 0x87, 0xee, 0x6b, 0x9d, 0x2f, 0x14, 0xeb, 0x8e, 0xc8, 0x24, 0x98, 0xf5, 0xcb, 0x82,
	0x77, 0xf3, 0x0a, 0x08, 0xcf, 0x2b, 0xe1, 0x4c, 0x5e, 0xac, 0x15, 0x0b, 0x71, 0x0b, 0x0a, 0x97,
	0x15, 0x10, 0x9e, 0xd3, 0x03, 0xe8, 0x1d, 0x49, 0xfb, 0xc6, 0xa4, 0x0b, 0xc5, 0x22, 0x5c, 0x72,
	0xb3, 0x2c, 0x38, 0x4d, 0x6a, 0xf6, 0x40, 0x67, 0xa9, 0x53, 0xd9, 0xca, 0x7d, 0x12, 0x57, 0x5e,
	0x3d, 0xf3, 0x2a, 0xcd, 0x52, 0xc7, 0x7a, 0x7f, 0xcc, 0x20, 0xfb, 0x6b, 0x06, 0x19, 0xbd, 0x07,
	0xc1, 0xa1, 0x74, 0x92, 0xf5, 0xf1, 0x7f, 0xd0, 0xb2, 0xe0, 0xbb, 0x4b, 0xe9, 0x64, 0xcb, 0xc5,
	0x9c, 0x3e, 0x86, 0xfe, 0x69, 0x9a, 0xe4, 0xd2, 0xad, 0x8d, 0x62, 0x80, 0xf2, 0xad, 0xb2, 0xe0,
	0xff, 0xdb, 0x06, 0xb6, 0x26, 0xae, 0x4d, 0xfa, 0x10, 0xa2, 0x93, 0x55, 0x55, 0x99, 0x65, 0x03,
	0xfc, 0xac, 0x37, 0xca, 0x82, 0x0f, 0xb5, 0x47, 0xad, 0x91, 0xc6, 0x1a, 0x7f, 0x25, 0x30, 0x6c,
	0x35, 0x2d, 0x94, 0x5d, 0x5f, 0x38, 0x3a, 0x86, 0x70, 0xfe, 0xf1, 0x85, 0xb4, 0xe7, 0x75, 0xdb,
	0xbe, 0x45, 0x24, 0xa2, 0x4e, 0xaa, 0x6a, 0x8e, 0xa4, 0x7d, 0x6b, 0xd5, 0x12, 0xab, 0x0e, 0x7c,
	0x35, 0x89, 0x47, 0xa2, 0xc9, 0xe8, 0x31, 0x84, 0xa7, 0x4e, 0xba, 0xb5, 0xc5, 0x9e, 0x77, 0x0f,
	0xf6, 0xfd, 0x13, 0x9b, 0xfe, 0x73, 0xe9, 0xd4, 0x1f, 0x5e, 0xf6, 0x37, 0x5a, 0xfc, 0x2d, 0xea,
	0x05, 0x63, 0x0e, 0x3b, 0x6d, 0x87, 0x86, 0xb0, 0x7d, 0xf2, 0x72, 0x6f, 0x8b, 0x46, 0xd0, 0x79,
	0x66, 0xcc, 0x1e, 0x99, 0xdd, 0xf9, 0xb2, 0x89, 0xc9, 0xb7, 0x4d, 0x4c, 0xbe, 0x6f, 0x62, 0xf2,
	0x63, 0x13, 0x93, 0xcf, 0x3f, 0xe3, 0xad, 0x77, 0x83, 0xd6, 0x93, 0x7f, 0x1f, 0xe2, 0xed, 0x8f,
	0x7e, 0x07, 0x00, 0x00, 0xff, 0xff, 0x80, 0x15, 0xda, 0xf6, 0x08, 0x03, 0x00, 0x00,
}

func (m *Transaction) Marshal() (dAtA []byte, err error) {
	size := m.Size()
	dAtA = make([]byte, size)
	n, err := m.MarshalToSizedBuffer(dAtA[:size])
	if err != nil {
		return nil, err
	}
	return dAtA[:n], nil
}

func (m *Transaction) MarshalTo(dAtA []byte) (int, error) {
	size := m.Size()
	return m.MarshalToSizedBuffer(dAtA[:size])
}

func (m *Transaction) MarshalToSizedBuffer(dAtA []byte) (int, error) {
	i := len(dAtA)
	_ = i
	var l int
	_ = l
	if m.XXX_unrecognized != nil {
		i -= len(m.XXX_unrecognized)
		copy(dAtA[i:], m.XXX_unrecognized)
	}
	if m.Options != 0 {
		i = encodeVarintTransaction(dAtA, i, uint64(m.Options))
		i--
		dAtA[i] = 0x58
	}
	if len(m.Signature) > 0 {
		i -= len(m.Signature)
		copy(dAtA[i:], m.Signature)
		i = encodeVarintTransaction(dAtA, i, uint64(len(m.Signature)))
		i--
		dAtA[i] = 0x52
	}
	if len(m.Data) > 0 {
		i -= len(m.Data)
		copy(dAtA[i:], m.Data)
		i = encodeVarintTransaction(dAtA, i, uint64(len(m.Data)))
		i--
		dAtA[i] = 0x4a
	}
	if m.GasLimit != 0 {
		i = encodeVarintTransaction(dAtA, i, uint64(m.GasLimit))
		i--
		dAtA[i] = 0x40
	}
	if m.GasPrice != 0 {
		i = encodeVarintTransaction(dAtA, i, uint64(m.GasPrice))
		i--
		dAtA[i] = 0x38
	}
	if len(m.Value) > 0 {
		i -= len(m.Value)
		copy(dAtA[i:], m.Value)
		i = encodeVarintTransaction(dAtA, i, uint64(len(m.Value)))
		i--
		dAtA[i] = 0x32
	}
	if m.Nonce != 0 {
		i = encodeVarintTransaction(dAtA, i, uint64(m.Nonce))
		i--
		dAtA[i] = 0x28
	}
	if len(m.ChainID) > 0 {
		i -= len(m.ChainID)
		copy(dAtA[i:], m.ChainID)
		i = encodeVarintTransaction(dAtA, i, uint64(len(m.ChainID)))
		i--
		dAtA[i] = 0x22
	}
	if m.Version != 0 {
		i = encodeVarintTransaction(dAtA, i, uint64(m.Version))
		i--
		dAtA[i] = 0x18
	}
	if len(m.TargetAddr) > 0 {
		i -= len(m.TargetAddr)
		copy(dAtA[i:], m.TargetAddr)
		i = encodeVarintTransaction(dAtA, i, uint64(len(m.TargetAddr)))
		i--
		dAtA[i] = 0x12
	}
	if len(m.FromAddr) > 0 {
		i -= len(m.FromAddr)
		copy(dAtA[i:], m.FromAddr)
		i = encodeVarintTransaction(dAtA, i, uint64(len(m.FromAddr)))
		i--
		dAtA[i] = 0xa
	}
	return len(dAtA) - i, nil
}

func (m *TransactionResult) Marshal() (dAtA []byte, err error) {
	size := m.Size()
	dAtA = make([]byte, size)
	n, err := m.MarshalToSizedBuffer(dAtA[:size])
	if err != nil {
		return nil, err
	}
	return dAtA[:n], nil
}

func (m *TransactionResult) MarshalTo(dAtA []byte) (int, error) {
	size := m.Size()
	return m.MarshalToSizedBuffer(dAtA[:size])
}

func (m *TransactionResult) MarshalToSizedBuffer(dAtA []byte) (int, error) {
	i := len(dAtA)
	_ = i
	var l int
	_ = l
	if m.XXX_unrecognized != nil {
		i -= len(m.XXX_unrecognized)
		copy(dAtA[i:], m.XXX_unrecognized)
	}
	if m.Status != 0 {
		i = encodeVarintTransaction(dAtA, i, uint64(m.Status))
		i--
		dAtA[i] = 0x18
	}
	if m.GasUsed != 0 {
		i = encodeVarintTransaction(dAtA, i, uint64(m.GasUsed))
		i--
		dAtA[i] = 0x10
	}
	if len(m.TxHash) > 0 {
		i -= len(m.TxHash)
		copy(dAtA[i:], m.TxHash)
		i = encodeVarintTransaction(dAtA, i, uint64(len(m.TxHash)))
		i--
		dAtA[i] = 0xa
	}
	return len(dAtA) - i, nil
}

func encodeVarintTransaction(dAtA []byte, offset int, v uint64) int {
	offset -= sovTransaction(v)
	base := offset
	for v >= 1<<7 {
		dAtA[offset] = uint8(v&0x7f | 0x80)
		v >>= 7
		offset++
	}
	dAtA[offset] = uint8(v)
	return base
}
func (m *Transaction) Size() (n int) {
	if m == nil {
		return 0
	}
	var l int
	_ = l
	l = len(m.FromAddr)
	if l > 0 {
		n += 1 + l + sovTransaction(uint64(l))
	}
	l = len(m.TargetAddr)
	if l > 0 {
		n += 1 + l + sovTransaction(uint64(l))
	}
	if m.Version != 0 {
		n += 1 + sovTransaction(uint64(m.Version))
	}
	l = len(m.ChainID)
	if l > 0 {
		n += 1 + l + sovTransaction(uint64(l))
	}
	if m.Nonce != 0 {
		n += 1 + sovTransaction(uint64(m.Nonce))
	}
	l = len(m.Value)
	if l > 0 {
		n += 1 + l + sovTransaction(uint64(l))
	}
	if m.GasPrice != 0 {
		n += 1 + sovTransaction(uint64(m.GasPrice))
	}
	if m.GasLimit != 0 {
		n += 1 + sovTransaction(uint64(m.GasLimit))
	}
	l = len(m.Data)
	if l > 0 {
		n += 1 + l + sovTransaction(uint64(l))
	}
	l = len(m.Signature)
	if l > 0 {
		n += 1 + l + sovTransaction(uint64(l))
	}
	if m.Options != 0 {
		n += 1 + sovTransaction(uint64(m.Options))
	}
	if m.XXX_unrecognized != nil {
		n += len(m.XXX_unrecognized)
	}
	return n
}

func (m *TransactionResult) Size() (n int) {
	if m == nil {
		return 0
	}
	var l int
	_ = l
	l = len(m.TxHash)
	if l > 0 {
		n += 1 + l + sovTransaction(uint64(l))
	}
	if m.GasUsed != 0 {
		n += 1 + sovTransaction(uint64(m.GasUsed))
	}
	if m.Status != 0 {
		n += 1 + sovTransaction(uint64(m.Status))
	}
	if m.XXX_unrecognized != nil {
		n += len(m.XXX_unrecognized)
	}
	return n
}

func sovTransaction(x uint64) (n int) {
	return (math_bits.Len64(x|1) + 6) / 7
}
func sozTransaction(x uint64) (n int) {
	return sovTransaction(uint64((x << 1) ^ uint64((int64(x) >> 63))))
}
func (m *Transaction) Unmarshal(dAtA []byte) error {
	l := len(dAtA)
	iNdEx := 0
	for iNdEx < l {
		preIndex := iNdEx
		var wire uint64
		for shift := uint(0); ; shift += 7 {
			if shift >= 64 {
				return ErrIntOverflowTransaction
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
			return fmt.Errorf("proto: Transaction: wiretype end group for non-group")
		}
		if fieldNum <= 0 {
			return fmt.Errorf("proto: Transaction: illegal tag %d (wire type %d)", fieldNum, wire)
		}
		switch fieldNum {
		case 1:
			if wireType != 2 {
				return fmt.Errorf("proto: wrong wireType = %d for field FromAddr", wireType)
			}
			var byteLen int
			for shift := uint(0); ; shift += 7 {
				if shift >= 64 {
					return ErrIntOverflowTransaction
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
				return ErrInvalidLengthTransaction
			}
			postIndex := iNdEx + byteLen
			if postIndex < 0 {
				return ErrInvalidLengthTransaction
			}
			if postIndex > l {
				return io.ErrUnexpectedEOF
			}
			m.FromAddr = append(m.FromAddr[:0], dAtA[iNdEx:postIndex]...)
			if m.FromAddr == nil {
				m.FromAddr = []byte{}
			}
			iNdEx = postIndex
		case 2:
			if wireType != 2 {
				return fmt.Errorf("proto: wrong wireType = %d for field TargetAddr", wireType)
			}
			var byteLen int
			for shift := uint(0); ; shift += 7 {
				if shift >= 64 {
					return ErrIntOverflowTransaction
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
				return ErrInvalidLengthTransaction
			}
			postIndex := iNdEx + byteLen
			if postIndex < 0 {
				return ErrInvalidLengthTransaction
			}
			if postIndex > l {
				return io.ErrUnexpectedEOF
			}
			m.TargetAddr = append(m.TargetAddr[:0], dAtA[iNdEx:postIndex]...)
			if m.TargetAddr == nil {
				m.TargetAddr = []byte{}
			}
			iNdEx = postIndex
		case 3:
			if wireType != 0 {
				return fmt.Errorf("proto: wrong wireType = %d for field Version", wireType)
			}
			m.Version = 0
			for shift := uint(0); ; shift += 7 {
				if shift >= 64 {
					return ErrIntOverflowTransaction
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
		case 4:
			if wireType != 2 {
				return fmt.Errorf("proto: wrong wireType = %d for field ChainID", wireType)
			}
			var byteLen int
			for shift := uint(0); ; shift += 7 {
				if shift >= 64 {
					return ErrIntOverflowTransaction
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
				return ErrInvalidLengthTransaction
			}
			postIndex := iNdEx + byteLen
			if postIndex < 0 {
				return ErrInvalidLengthTransaction
			}
			if postIndex > l {
				return io.ErrUnexpectedEOF
			}
			m.ChainID = append(m.ChainID[:0], dAtA[iNdEx:postIndex]...)
			if m.ChainID == nil {
				m.ChainID = []byte{}
			}
			iNdEx = postIndex
		case 5:
			if wireType != 0 {
				return fmt.Errorf("proto: wrong wireType = %d for field Nonce", wireType)
			}
			m.Nonce = 0
			for shift := uint(0); ; shift += 7 {
				if shift >= 64 {
					return ErrIntOverflowTransaction
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
		case 6:
			if wireType != 2 {
				return fmt.Errorf("proto: wrong wireType = %d for field Value", wireType)
			}
			var byteLen int
			for shift := uint(0); ; shift += 7 {
				if shift >= 64 {
					return ErrIntOverflowTransaction
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
				return ErrInvalidLengthTransaction
			}
			postIndex := iNdEx + byteLen
			if postIndex < 0 {
				return ErrInvalidLengthTransaction
			}
			if postIndex > l {
				return io.ErrUnexpectedEOF
			}
			m.Value = append(m.Value[:0], dAtA[iNdEx:postIndex]...)
			if m.Value == nil {
				m.Value = []byte{}
			}
			iNdEx = postIndex
		case 7:
			if wireType != 0 {
				return fmt.Errorf("proto: wrong wireType = %d for field GasPrice", wireType)
			}
			m.GasPrice = 0
			for shift := uint(0); ; shift += 7 {
				if shift >= 64 {
					return ErrIntOverflowTransaction
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
		case 8:
			if wireType != 0 {
				return fmt.Errorf("proto: wrong wireType = %d for field GasLimit", wireType)
			}
			m.GasLimit = 0
			for shift := uint(0); ; shift += 7 {
				if shift >= 64 {
					return ErrIntOverflowTransaction
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
		case 9:
			if wireType != 2 {
				return fmt.Errorf("proto: wrong wireType = %d for field Data", wireType)
			}
			var byteLen int
			for shift := uint(0); ; shift += 7 {
				if shift >= 64 {
					return ErrIntOverflowTransaction
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
				return ErrInvalidLengthTransaction
			}
			postIndex := iNdEx + byteLen
			if postIndex < 0 {
				return ErrInvalidLengthTransaction
			}
			if postIndex > l {
				return io.ErrUnexpectedEOF
			}
			m.Data = append(m.Data[:0], dAtA[iNdEx:postIndex]...)
			if m.Data == nil {
				m.Data = []byte{}
			}
			iNdEx = postIndex
		case 10:
			if wireType != 2 {
				return fmt.Errorf("proto: wrong wireType = %d for field Signature", wireType)
			}
			var byteLen int
			for shift := uint(0); ; shift += 7 {
				if shift >= 64 {
					return ErrIntOverflowTransaction
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
				return ErrInvalidLengthTransaction
			}
			postIndex := iNdEx + byteLen
			if postIndex < 0 {
				return ErrInvalidLengthTransaction
			}
			if postIndex > l {
				return io.ErrUnexpectedEOF
			}
			m.Signature = append(m.Signature[:0], dAtA[iNdEx:postIndex]...)
			if m.Signature == nil {
				m.Signature = []byte{}
			}
			iNdEx = postIndex
		case 11:
			if wireType != 0 {
				return fmt.Errorf("proto: wrong wireType = %d for field Options", wireType)
			}
			m.Options = 0
			for shift := uint(0); ; shift += 7 {
				if shift >= 64 {
					return ErrIntOverflowTransaction
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
			skippy, err := skipTransaction(dAtA[iNdEx:])
			if err != nil {
				return err
			}
			if (skippy < 0) || (iNdEx+skippy) < 0 {
				return ErrInvalidLengthTransaction
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
func (m *TransactionResult) Unmarshal(dAtA []byte) error {
	l := len(dAtA)
	iNdEx := 0
	for iNdEx < l {
		preIndex := iNdEx
		var wire uint64
		for shift := uint(0); ; shift += 7 {
			if shift >= 64 {
				return ErrIntOverflowTransaction
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
			return fmt.Errorf("proto: TransactionResult: wiretype end group for non-group")
		}
		if fieldNum <= 0 {
			return fmt.Errorf("proto: TransactionResult: illegal tag %d (wire type %d)", fieldNum, wire)
		}
		switch fieldNum {
		case 1:
			if wireType != 2 {
				return fmt.Errorf("proto: wrong wireType = %d for field TxHash", wireType)
			}
			var byteLen int
			for shift := uint(0); ; shift += 7 {
				if shift >= 64 {
					return ErrIntOverflowTransaction
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
				return ErrInvalidLengthTransaction
			}
			postIndex := iNdEx + byteLen
			if postIndex < 0 {
				return ErrInvalidLengthTransaction
			}
			if postIndex > l {
				return io.ErrUnexpectedEOF
			}
			m.TxHash = append(m.TxHash[:0], dAtA[iNdEx:postIndex]...)
			if m.TxHash == nil {
				m.TxHash = []byte{}
			}
			iNdEx = postIndex
		case 2:
			if wireType != 0 {
				return fmt.Errorf("proto: wrong wireType = %d for field GasUsed", wireType)
			}
			m.GasUsed = 0
			for shift := uint(0); ; shift += 7 {
				if shift >= 64 {
					return ErrIntOverflowTransaction
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
		case 3:
			if wireType != 0 {
				return fmt.Errorf("proto: wrong wireType = %d for field Status", wireType)
			}
			m.Status = 0
			for shift := uint(0); ; shift += 7 {
				if shift >= 64 {
					return ErrIntOverflowTransaction
				}
				if iNdEx >= l {
					return io.ErrUnexpectedEOF
				}
				b := dAtA[iNdEx]
				iNdEx++
				m.Status |= TransactionResult_ResultStatus(b&0x7F) << shift
				if b < 0x80 {
					break
				}
			}
		default:
			iNdEx = preIndex
			skippy, err := skipTransaction(dAtA[iNdEx:])
			if err != nil {
				return err
			}
			if (skippy < 0) || (iNdEx+skippy) < 0 {
				return ErrInvalidLengthTransaction
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
func skipTransaction(dAtA []byte) (n int, err error) {
	l := len(dAtA)
	iNdEx := 0
	depth := 0
	for iNdEx < l {
		var wire uint64
		for shift := uint(0); ; shift += 7 {
			if shift >= 64 {
				return 0, ErrIntOverflowTransaction
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
					return 0, ErrIntOverflowTransaction
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
					return 0, ErrIntOverflowTransaction
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
				return 0, ErrInvalidLengthTransaction
			}
			iNdEx += length
		case 3:
			depth++
		case 4:
			if depth == 0 {
				return 0, ErrUnexpectedEndOfGroupTransaction
			}
			depth--
		case 5:
			iNdEx += 4
		default:
			return 0, fmt.Errorf("proto: illegal wireType %d", wireType)
		}
		if iNdEx < 0 {
			return 0, ErrInvalidLengthTransaction
		}
		if depth == 0 {
			return iNdEx, nil
		}
	}
	return 0, io.ErrUnexpectedEOF
}

var (
	ErrInvalidLengthTransaction        = fmt.Errorf("proto: negative length found during unmarshaling")
	ErrIntOverflowTransaction          = fmt.Errorf("proto: integer overflow")
	ErrUnexpectedEndOfGroupTransaction = fmt.Errorf("proto: unexpected end of group")
)
