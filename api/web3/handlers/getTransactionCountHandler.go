package handlers

import (
	"encoding/json"
	"github.com/TopiaNetwork/topia/api/servant"
	"github.com/TopiaNetwork/topia/api/web3/types"
	types2 "github.com/TopiaNetwork/topia/api/web3/types/hexutil"
	tpcrtypes "github.com/TopiaNetwork/topia/crypt/types"
	"strconv"
)

func GetTransactionCountHandler(parmas interface{}, apiServant interface{}) interface{} {
	args := parmas.(*types.GetTransactionCountRequestType)
	servant := apiServant.(servant.APIServant)

	height, _ := strconv.ParseUint(args.Height, 10, 64)
	count, err := servant.GetTransactionCount(tpcrtypes.NewFromString(args.Address.String()), height)
	if err != nil {
		return types.ErrorMessage(err)
	}
	txCount := types2.Uint(count)

	enc, err := json.Marshal(txCount)
	if err != nil {
		return types.ErrorMessage(err)
	}
	return &types.JsonrpcMessage{Result: enc}
}
