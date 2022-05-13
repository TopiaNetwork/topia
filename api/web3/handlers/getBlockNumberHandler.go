package handlers

import (
	"encoding/json"
	"github.com/TopiaNetwork/topia/api/servant"
	"github.com/TopiaNetwork/topia/api/web3/types"
	types2 "github.com/TopiaNetwork/topia/api/web3/types/hexutil"
)

func GetBlockNumberHandler(parmas interface{}, apiServant interface{}) interface{} {
	servant := apiServant.(servant.APIServant)

	block, err := servant.GetLatestBlock()
	if err != nil {
		return types.ErrorMessage(err)
	}
	height := block.GetHead().GetHeight()

	enc, err := json.Marshal(types2.Uint64(height))
	if err != nil {
		return types.ErrorMessage(err)
	}
	return &types.JsonrpcMessage{Result: enc}
}
