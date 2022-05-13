package handlers

import (
	"encoding/json"
	"github.com/TopiaNetwork/topia/api/servant"
	"github.com/TopiaNetwork/topia/api/web3/convert"
	"github.com/TopiaNetwork/topia/api/web3/types"
)

func GetTransactionByHashHandler(parmas interface{}, apiServant interface{}) interface{} {
	args := parmas.(*types.GetTransactionByHashRequestType)
	servant := apiServant.(servant.APIServant)

	transaction, err := servant.GetTransactionByHash(args.TxHash)
	if err != nil {
		return types.ErrorMessage(err)
	}
	ethTransaction, err := convert.TopiaTransactionToEthTransaction(*transaction, servant)
	if err != nil {
		return types.ErrorMessage(err)
	}
	enc, err := json.Marshal(ethTransaction)
	if err != nil {
		return types.ErrorMessage(err)
	}
	return &types.JsonrpcMessage{Result: enc}
}
