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
start cipher  |  requestIdSize  |  serverMethodSize  |  AuthCodeSize  |  ErrMsgSize  |  payloadSize    ||    msgType  |  requestId  |  serverMethod  |  AuthCode  |  ErrMsg  |  payload
    0x05      |        4        |         4          |       4        |       4      |       4         ||   x(1 byte) |  x(8 bytes) |       x        |     x      |     x    |     x

*/

type Message struct {
	Header     *Header
	MsgType    MsgType
	RequestId  string // also used as subID when MsgType == MsgSubscribe
	MethodName string // also used as eventName when MsgType is related to subscribe
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

type MsgType byte

const (
	MsgCall              MsgType = 1
	MsgCallResp          MsgType = 2
	MsgSubscribeReq      MsgType = 3
	MsgSubscribeReqAck   MsgType = 4
	MsgUnsubscribeReq    MsgType = 5
	MsgUnsubscribeReqAck MsgType = 6
	MsgSubscribe         MsgType = 7 // subscribed msg
)

type Errtype int

const (
	NoErr                 Errtype = 0
	ErrMethodNotFound     Errtype = 1
	ErrAuthFailed         Errtype = 2
	ErrIllegalArgument    Errtype = 3
	ErrServiceReturnError Errtype = 4 // Error return by called service

	ErrNoSuchEvent       Errtype = 5
	ErrExceedMaxSubLimit Errtype = 6 // client is trying to sub one event for more than ClientMaxSubsToOneEvent
	ErrInvalidSubID      Errtype = 7
	ErrNoSuchSub         Errtype = 8
)

type ErrMsg struct {
	Errtype   Errtype `json:"err_type"`
	ErrCode   int     `json:"err_code"`
	ErrString string  `json:"err_string"`
	Data      string  `json:"data"`
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

	bodyLen := 1 + header.RequestIdSize + header.MethodNameSize + header.AuthCodeSize + header.ErrMsgSize + header.PayloadSize // 1 is for msgType
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
	if header.Sc != StartCipher {
		return nil, errors.New("wrong StartCipher")
	}
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
	endI := st + 1
	result.MsgType = MsgType(data[st])

	st = endI
	endI = st + header.RequestIdSize
	result.RequestId = string(data[st:endI])

	st = endI
	endI = st + header.MethodNameSize
	result.MethodName = string(data[st:endI])

	st = endI
	endI = st + header.AuthCodeSize
	result.AuthCode = string(data[st:endI])

	st = endI
	endI = st + header.ErrMsgSize
	err := json.Unmarshal(data[st:endI], &result.ErrMsg)
	if err != nil {
		return nil, err
	}

	st = endI
	endI = st + header.PayloadSize
	payload := make([]byte, header.PayloadSize)
	copy(payload, data[st:endI])
	result.Payload = payload

	return &result, nil
}

// EncodeMessage 基础编码
func EncodeMessage(msgType MsgType, requestId string, methodName string, authCode string, errMsg *ErrMsg, payload []byte) (data []byte, err error) {

	errMsgBytes, err := json.Marshal(errMsg)
	if err != nil {
		return nil, err
	}

	bufSize := HeadSize + 1 + len(requestId) + len(methodName) + len(authCode) + len(errMsgBytes) + len(payload) // 1 is for msgType
	buf := make([]byte, bufSize)

	buf[0] = StartCipher
	binary.BigEndian.PutUint32(buf[1:5], uint32(len(requestId)))
	binary.BigEndian.PutUint32(buf[5:9], uint32(len(methodName)))
	binary.BigEndian.PutUint32(buf[9:13], uint32(len(authCode)))
	binary.BigEndian.PutUint32(buf[13:17], uint32(len(errMsgBytes)))
	binary.BigEndian.PutUint32(buf[17:21], uint32(len(payload)))

	st := HeadSize
	endI := st + 1
	copy(buf[st:endI], []byte{byte(msgType)})

	st = endI
	endI = st + len(requestId)
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
