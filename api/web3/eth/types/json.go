package types

import "encoding/json"

const (
	vsn = "2.0"
)

var Null = json.RawMessage("null")

type JsonrpcMessage struct {
	Version string          `json:"jsonrpc,omitempty"`
	ID      json.RawMessage `json:"id,omitempty"`
	Method  string          `json:"method,omitempty"`
	Params  json.RawMessage `json:"params,omitempty"`
	Error   *jsonError      `json:"error,omitempty"`
	Result  json.RawMessage `json:"result,omitempty"`
}

func (msg *JsonrpcMessage) Response(result interface{}) *JsonrpcMessage {
	enc, err := json.Marshal(result)
	if err != nil {
		return msg.ErrorResponse(err)
	}
	return &JsonrpcMessage{Version: vsn, ID: msg.ID, Result: enc}
}

func (msg *JsonrpcMessage) hasValidID() bool {
	return len(msg.ID) > 0 && msg.ID[0] != '{' && msg.ID[0] != '['
}

func (msg *JsonrpcMessage) ErrorResponse(err error) *JsonrpcMessage {
	//resp := errorMessage(err)
	//resp.ID = msg.ID
	return nil
}

func ErrorMessage(err error) *JsonrpcMessage {
	msg := &JsonrpcMessage{Version: vsn, ID: Null, Error: &jsonError{
		Code:    defaultErrorCode,
		Message: err.Error(),
	}}
	ec, ok := err.(Error)
	if ok {
		msg.Error.Code = ec.ErrorCode()
	}
	de, ok := err.(DataError)
	if ok {
		msg.Error.Data = de.ErrorData()
	}
	return msg
}

func (msg *JsonrpcMessage) String() string {
	b, _ := json.Marshal(msg)
	return string(b)
}

type jsonError struct {
	Code    int         `json:"code"`
	Message string      `json:"message"`
	Data    interface{} `json:"data,omitempty"`
}
