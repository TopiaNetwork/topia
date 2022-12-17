package rpc

import (
	"encoding/binary"
	"encoding/json"
	"errors"
	"io"
	"log"
)

type RequestType byte

const (
	Request RequestType = iota
	Response
	HeartBeat
)

const StartCipher = 0x05 // 起始符
const HeadSize = 21

/*
协议设计
--------------------------------------------------------------------------------------------------------------------------------------------------
start cipher  |  requestIdSize  |  serverMethodSize  |  AuthCodeSize  |  ErrMsgSize  |  payloadSize    ||    requestId  |  serverMethod  |  AuthCode  |  ErrMsg  |  payload
    0x05      |        4        |         4          |       4        |       4      |       4         ||    x(8 bytes) |       x        |     x      |     x    |     x

*/

type Message struct {
	Header     *Header
	RequestId  string
	MethodName string
	AuthCode   string
	ErrMsg     ErrMsg
	Payload    []byte
}

type Header struct {
	Sc             byte
	RequestIdSize  uint32
	MethodNameSize uint32
	AuthCodeSize   uint32
	ErrMsgSize     uint32
	PayloadSize    uint32
}

type Errtype int

const (
	NoErr                 Errtype = 0
	ErrMethodNotFound     Errtype = 1
	ErrAuthFailed         Errtype = 2
	ErrIllegalArgument    Errtype = 3
	ErrServiceReturnError Errtype = 4 // Error return by called service
)

type ErrMsg struct {
	Errtype   Errtype `json:"err_type"`
	ErrCode   int     `json:"err_code"`
	ErrString string  `json:"err_string"`
}

func IODecodeMessage(r io.Reader) (*Message, error) {
	headerByte := make([]byte, HeadSize)
	// 读取标志位
	_, err := io.ReadFull(r, headerByte[:1])
	if err != nil {
		return nil, err
	}

	if headerByte[0] != StartCipher {
		log.Println(headerByte)
		return nil, errors.New("wrong StartCipher")
	}

	// 读取剩下的
	_, err = io.ReadFull(r, headerByte[1:])
	if err != nil {
		return nil, err
	}

	// 解析 header
	header, err := DecodeHeader(headerByte)
	if err != nil {
		return nil, err
	}

	bodyLen := header.RequestIdSize + header.MethodNameSize + header.AuthCodeSize + header.ErrMsgSize + header.PayloadSize
	bodyData := make([]byte, bodyLen)
	_, err = io.ReadFull(r, bodyData)
	if err != nil {
		return nil, err
	}

	msg, err := DecodeMessageV2(bodyData, header, 0)
	if err != nil {
		return nil, err
	}

	return msg, nil
}

func DecodeHeader(data []byte) (*Header, error) {
	var header Header
	header.Sc = data[0]
	header.RequestIdSize = binary.BigEndian.Uint32(data[1:5])
	header.MethodNameSize = binary.BigEndian.Uint32(data[5:9])
	header.AuthCodeSize = binary.BigEndian.Uint32(data[9:13])
	header.ErrMsgSize = binary.BigEndian.Uint32(data[13:17])
	header.PayloadSize = binary.BigEndian.Uint32(data[17:21])

	return &header, nil
}

// DecodeMessage 完整Decode
func DecodeMessage(data []byte) (*Message, error) {
	header, err := DecodeHeader(data)
	if err != nil {
		return nil, err
	}
	return DecodeMessageV2(data, header, HeadSize)
}

func DecodeMessageV2(data []byte, header *Header, headSize uint32) (*Message, error) {
	var result Message
	result.Header = header
	var st uint32 = headSize
	endI := st + header.RequestIdSize
	les := endI - st
	RequestId := make([]byte, les)
	copy(RequestId, data[st:endI])
	result.RequestId = string(RequestId)

	st = endI
	endI = st + header.MethodNameSize
	les = endI - st
	MethodName := make([]byte, les)
	copy(MethodName, data[st:endI])
	result.MethodName = string(MethodName)

	st = endI
	endI = st + header.AuthCodeSize
	les = endI - st
	AuthCode := make([]byte, les)
	copy(AuthCode, data[st:endI])
	result.AuthCode = string(AuthCode)

	st = endI
	endI = st + header.ErrMsgSize
	les = endI - st
	err := json.Unmarshal(data[st:endI], &result.ErrMsg)
	if err != nil {
		return nil, err
	}

	st = endI
	endI = st + header.PayloadSize
	les = endI - st
	payload := make([]byte, les)
	copy(payload, data[st:endI])
	result.Payload = payload

	return &result, nil
}

// EncodeMessage 基础编码
func EncodeMessage(requestId string, methodName string, authCode string, errMsg *ErrMsg, payload []byte) (data []byte, err error) {

	errMsgBytes, err := json.Marshal(errMsg)
	if err != nil {
		return nil, err
	}

	bufSize := HeadSize + len(requestId) + len(methodName) + len(authCode) + len(errMsgBytes) + len(payload)
	buf := make([]byte, bufSize)

	buf[0] = StartCipher
	binary.BigEndian.PutUint32(buf[1:5], uint32(len(requestId)))
	binary.BigEndian.PutUint32(buf[5:9], uint32(len(methodName)))
	binary.BigEndian.PutUint32(buf[9:13], uint32(len(authCode)))
	binary.BigEndian.PutUint32(buf[13:17], uint32(len(errMsgBytes)))
	binary.BigEndian.PutUint32(buf[17:21], uint32(len(payload)))

	st := HeadSize
	endI := st + len(requestId)
	copy(buf[st:endI], requestId)

	st = endI
	endI = st + len(methodName)
	copy(buf[st:endI], methodName)

	st = endI
	endI = st + len(authCode)
	copy(buf[st:endI], authCode)

	st = endI
	endI = st + len(errMsgBytes)
	copy(buf[st:endI], errMsgBytes)

	st = endI
	endI = st + len(payload)
	copy(buf[st:endI], payload)

	return buf, nil
}
