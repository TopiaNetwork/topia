package handlers

import (
	"encoding/json"
	"github.com/TopiaNetwork/topia/api/servant"
	"github.com/TopiaNetwork/topia/api/web3/convert"
	"github.com/TopiaNetwork/topia/api/web3/types"
)

func GetBlockByHashHandler(parmas interface{}, apiServant interface{}) interface{} {
	args := parmas.(*types.GetBlockByHashRequestType)
	servant := apiServant.(servant.APIServant)

	block, err := servant.GetBlockByHash(args.BlockHash)
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
