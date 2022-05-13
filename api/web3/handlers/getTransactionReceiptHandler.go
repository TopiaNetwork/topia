package handlers

import (
	"encoding/json"
	"github.com/TopiaNetwork/topia/api/servant"
	"github.com/TopiaNetwork/topia/api/web3/convert"
	"github.com/TopiaNetwork/topia/api/web3/types"
)

func GetTransactionReceiptHandler(parmas interface{}, apiServant interface{}) interface{} {
	args := parmas.(*types.GetTransactionReceiptRequestType)
	servant := apiServant.(servant.APIServant)
	result, err := servant.GetTransactionResultByHash(args.TxHash.Hex())
	if err != nil {
		return types.ErrorMessage(err)
	}
	ethReceipt := convert.ConvertTopiaResultToEthReceipt(*result, servant)

	enc, err := json.Marshal(ethReceipt)
	if err != nil {
		return types.ErrorMessage(err)
	}
	return &types.JsonrpcMessage{Result: enc}
}
