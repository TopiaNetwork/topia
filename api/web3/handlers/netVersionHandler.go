package handlers

import (
	"encoding/json"
	"github.com/TopiaNetwork/topia/api/web3/types"
	"math/big"
)

func NetVersionHandler(parmas interface{}, apiServant interface{}) interface{} {
	enc, err := json.Marshal(big.NewInt(9))
	if err != nil {
		return types.ErrorMessage(err)
	}
	return &types.JsonrpcMessage{Result: enc}
}
