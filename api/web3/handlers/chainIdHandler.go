package handlers

import (
	"encoding/json"
	"github.com/TopiaNetwork/topia/api/servant"
	"github.com/TopiaNetwork/topia/api/web3/types"
	types2 "github.com/TopiaNetwork/topia/api/web3/types/hexutil"
	"math/big"
)

func ChainIdHandler(parmas interface{}, apiServant interface{}) interface{} {
	servant := apiServant.(servant.APIServant)
	chainId, _ := new(big.Int).SetString(string(servant.ChainID()), 10)
	chId := (*types2.Big)(chainId)
	enc, err := json.Marshal(chId)
	if err != nil {
		return types.ErrorMessage(err)
	}
	return &types.JsonrpcMessage{Result: enc}
}
