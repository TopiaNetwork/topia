package handlers

import (
	"encoding/json"
	"github.com/TopiaNetwork/topia/api/servant"
	"github.com/TopiaNetwork/topia/api/web3/types"
)

func FeeHistoryHandler(parmas interface{}, apiServant interface{}) interface{} {
	args := parmas.(*types.FeeHistoryRequestType)
	servant := apiServant.(servant.APIServant)

	feeHistory := types.FeeHistory(args, servant)

	enc, err := json.Marshal(feeHistory)
	if err != nil {
		return types.ErrorMessage(err)
	}
	return &types.JsonrpcMessage{Result: enc}
}
