package handlers

import (
	"encoding/json"
	"github.com/TopiaNetwork/topia/api/servant"
	"github.com/TopiaNetwork/topia/api/web3/types"
	types2 "github.com/TopiaNetwork/topia/api/web3/types/hexutil"
	"math/big"
)

func EstimateGasHandler(parmas interface{}, apiServant interface{}) interface{} {
	args := parmas.(*types.EstimateGasRequestType)
	servant := apiServant.(servant.APIServant)

	tx := types.ConstructGasTransaction(*args)
	estimateGas, err := servant.EstimateGas(tx)
	if err != nil {
		return types.ErrorMessage(err)
	}
	gas := types2.Uint64(new(big.Int).SetBytes(estimateGas.Bytes()).Uint64())

	enc, err := json.Marshal(gas)
	if err != nil {
		return types.ErrorMessage(err)
	}
	return &types.JsonrpcMessage{Result: enc}
}
