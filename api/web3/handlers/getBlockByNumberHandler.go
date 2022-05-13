package handlers

import (
	"encoding/json"
	"github.com/TopiaNetwork/topia/api/servant"
	"github.com/TopiaNetwork/topia/api/web3/convert"
	"github.com/TopiaNetwork/topia/api/web3/types"
	"strconv"
)

func GetBlockByNumberHandler(parmas interface{}, apiServant interface{}) interface{} {
	args := parmas.(*types.GetBlockByNumberRequestType)
	servant := apiServant.(servant.APIServant)

	var height uint64
	number := args.Height
	if value, ok := number.(string); ok {
		height, _ = strconv.ParseUint(value[2:], 16, 64)
	} else if value, ok := number.(float64); ok {
		height = uint64(value)
	}

	block, err := servant.GetBlockByHeight(height)
	if err != nil {
		return types.ErrorMessage(err)
	}

	ethBlock := convert.ConvertTopiaBlockToEthBlock(block)
	enc, err := json.Marshal(ethBlock)
	if err != nil {
		return types.ErrorMessage(err)
	}
	return &types.JsonrpcMessage{Result: enc}
}
