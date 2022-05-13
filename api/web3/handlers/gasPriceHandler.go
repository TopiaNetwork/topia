package handlers

import (
	"encoding/json"
	"github.com/TopiaNetwork/topia/api/servant"
	"github.com/TopiaNetwork/topia/api/web3/types"
	types2 "github.com/TopiaNetwork/topia/api/web3/types/hexutil"
)

func GasPriceHandler(parmas interface{}, apiServant interface{}) interface{} {
	gasPrice, err := types.EstimateTipFee(apiServant.(servant.APIServant))
	if err != nil {
		return types.ErrorMessage(err)
	}

	enc, err := json.Marshal((*types2.Big)(gasPrice))
	if err != nil {
		return types.ErrorMessage(err)
	}
	return &types.JsonrpcMessage{Result: enc}
}
