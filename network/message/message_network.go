package message

import (
	"encoding/json"
	"fmt"
)

type ResponseData struct {
	Data   []byte
	ErrMsg string
}

type SendResponse struct {
	NodeID   string
	RespData []byte
	Err      error
}

func ParseSendResp(respList []SendResponse) (remoteNodes []string, respBytes [][]byte, respErrs []string) {
	remoteNodes = make([]string, len(respList))
	respErrs = make([]string, len(respList))
	respBytes = make([][]byte, len(respList))
	for i, resp := range respList {
		remoteNodes[i] = resp.NodeID
		errMsg := func() string {
			respErrI := fmt.Sprintf("resp%d, nodeID %s:", i, resp.NodeID)
			if resp.Err != nil {
				respErrI += resp.Err.Error()
				return respErrI
			}
			var respData ResponseData
			err := json.Unmarshal(resp.RespData, &respData)
			if err != nil {
				respErrI += err.Error()
				return respErrI
			}
			if respData.ErrMsg != "" {
				respErrI += respData.ErrMsg
				return respErrI
			}

			respBytes[i] = respData.Data

			return ""
		}()

		respErrs[i] = errMsg
	}

	return
}
