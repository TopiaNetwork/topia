package message

type SendResponse struct {
	NodeID   string
	RespData []byte
	Err      error
}
