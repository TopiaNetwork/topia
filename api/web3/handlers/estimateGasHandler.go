package handlers

import (
	"encoding/json"
	"github.com/TopiaNetwork/topia/api/servant"
	"github.com/TopiaNetwork/topia/api/web3/types"
)

//具体的api调用过程
func EstimateGasHandler(parmas interface{}, apiServant interface{}) interface{} {
	//获取参数
	//TODO：参数格式校验
	args := parmas.(*types.EstimateGasRequestType)
	servant := apiServant.(servant.APIServant)
	//调用方法
	tx := types.ConstructGasTransaction(*args)
	estimateGas, err := servant.EstimateGas(tx)
	//3.进行类型转换，将该api的返回值包装为JsonRpcMessage
	enc, err := json.Marshal(estimateGas)
	if err != nil {
		//TODO:
		return nil //msg.ErrorResponse(err)
	}
	return &types.JsonrpcMessage{Result: enc}
}
