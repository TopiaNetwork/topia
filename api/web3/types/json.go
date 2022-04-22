package types

import "encoding/json"

const (
	vsn              = "2.0"
	defaultErrorCode = -32000
)

var null = json.RawMessage("null")

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

func (msg *JsonrpcMessage) ErrorResponse(err error) *JsonrpcMessage {
	resp := errorMessage(err)
	resp.ID = msg.ID
	return resp
}

func errorMessage(err error) *JsonrpcMessage {
	msg := &JsonrpcMessage{Version: vsn, ID: null, Error: &jsonError{
		Code:    defaultErrorCode,
		Message: err.Error(),
	}}
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

type invalidParamsError struct{ message string }

func (e *invalidParamsError) ErrorCode() int { return -32602 }

func (e *invalidParamsError) Error() string { return e.message }

func ErrorResponse(err error) *JsonrpcMessage {
	err = &invalidParamsError{err.Error()}
	msg := &JsonrpcMessage{Version: vsn, ID: null, Error: &jsonError{
		Code:    defaultErrorCode,
		Message: err.Error(),
	}}
	return msg
}
