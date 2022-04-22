package handlers

import (
	"encoding/json"
	"github.com/TopiaNetwork/topia/api/servant"
	"github.com/TopiaNetwork/topia/api/web3/types"
)

//具体的api调用过程
func GetTransactionByHashHandler(parmas interface{}, apiServant interface{}) interface{} {
	//获取参数
	args := parmas.(*types.GetTransactionByHashRequestType)
	servant := apiServant.(servant.APIServant)
	//调用方法
	transaction, err := servant.GetTransactionByHash(args.TxHash)
	//3.进行类型转换，将该api的返回值包装为JsonRpcMessage
	ethTransaction := types.ConstructGetTransactionByhashResponseType(*transaction)
	enc, err := json.Marshal(ethTransaction)
	if err != nil {
		return nil //msg.ErrorResponse(err)
	}
	return &types.JsonrpcMessage{Result: enc}
}
