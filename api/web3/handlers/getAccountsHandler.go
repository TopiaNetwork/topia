package handlers

import (
	"encoding/json"
	"github.com/TopiaNetwork/topia/api/web3/types"
)

func GetAccountsHandler(parmas interface{}, apiServant interface{}) interface{} {
	var accounts []string
	accounts = append(accounts, "0x3F1B3C065aeE8cA34c47fa84aAC3024E95a2E6D9")

	enc, err := json.Marshal(accounts)
	if err != nil {
		//TODO:
		return nil //msg.ErrorResponse(err)
	}
	return &types.JsonrpcMessage{Result: enc}
}
