package message

type ResponseData struct {
	Data   []byte
	ErrMsg string
}

type SendResponse struct {
	NodeID   string
	RespData []byte
	Err      error
}
