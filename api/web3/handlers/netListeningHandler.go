package handlers

import (
	"encoding/json"
	"github.com/TopiaNetwork/topia/api/web3/types"
)

func NetListeningHandler(parmas interface{}, apiServant interface{}) interface{} {
	enc, err := json.Marshal(true)
	if err != nil {
		return types.ErrorMessage(err)
	}
	return &types.JsonrpcMessage{Result: enc}
}
