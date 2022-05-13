package handlers

import (
	"context"
	"encoding/json"
	"github.com/TopiaNetwork/topia/api/servant"
	"github.com/TopiaNetwork/topia/api/web3/types"
	types2 "github.com/TopiaNetwork/topia/api/web3/types/hexutil"
	"github.com/TopiaNetwork/topia/codec"
	"github.com/TopiaNetwork/topia/transaction/universal"
)

func CallHandler(parmas interface{}, apiServant interface{}) interface{} {
	args := parmas.(*types.CallRequestType)
	servant := apiServant.(servant.APIServant)

	ctx := context.Background()
	tx := types.ConstructTransaction(*args)
	callResult, err := servant.ExecuteTxSim(ctx, tx)
	if err != nil {
		return types.ErrorMessage(err)
	}
	var transactionResultUniversal universal.TransactionResultUniversal
	marshaler := codec.CreateMarshaler(codec.CodecType_PROTO)
	_ = marshaler.Unmarshal(callResult.GetData().GetSpecification(), &transactionResultUniversal)
	result := transactionResultUniversal.GetData()
	var re types2.Bytes
	re = result
	enc, err := json.Marshal(re)
	if err != nil {
		return types.ErrorMessage(err)
	}
	return &types.JsonrpcMessage{Result: enc}
}
