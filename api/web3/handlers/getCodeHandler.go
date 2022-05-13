package handlers

import (
	"encoding/json"
	"github.com/TopiaNetwork/topia/api/servant"
	"github.com/TopiaNetwork/topia/api/web3/types"
	types2 "github.com/TopiaNetwork/topia/api/web3/types/hexutil"
	tpcrtypes "github.com/TopiaNetwork/topia/crypt/types"
	"strconv"
)

func GetCodeHandler(parmas interface{}, apiServant interface{}) interface{} {
	args := parmas.(*types.GetCodeRequestType)
	servant := apiServant.(servant.APIServant)

	var height uint64
	number := args.Height
	if value, ok := number.(string); ok {
		height, _ = strconv.ParseUint(value[2:], 16, 64)
	} else if value, ok := number.(float64); ok {
		height = uint64(value)
	}

	code, err := servant.GetContractCode(tpcrtypes.NewFromString(args.Address), height)
	if err != nil {
		return types.ErrorMessage(err)
	}

	enc, err := json.Marshal(types2.Bytes(code))
	if err != nil {
		return types.ErrorMessage(err)
	}
	return &types.JsonrpcMessage{Result: enc}
}
