package handlers

import (
	"context"
	"encoding/json"
	"github.com/TopiaNetwork/topia/api/servant"
	"github.com/TopiaNetwork/topia/api/web3/types"
)

//具体的api调用过程
func CallHandler(parmas interface{}, apiServant interface{}) interface{} {
	//获取参数
	args := parmas.(*types.CallRequestType)
	servant := apiServant.(servant.APIServant)
	//调用方法
	ctx := context.Background()
	tx := types.ConstructTransaction(*args)
	callResult, err := servant.ExecuteTxSim(ctx, tx)
	//3.进行类型转换，将该api的返回值包装为JsonRpcMessage
	ethResult := types.ConstructGetTransactionReceiptResponseType(*callResult)
	enc, err := json.Marshal(ethResult)
	if err != nil {
		//TODO:
		return nil //msg.ErrorResponse(err)
	}
	return &types.JsonrpcMessage{Result: enc}
}
