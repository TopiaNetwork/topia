package handlers

import (
	"encoding/json"
	"github.com/TopiaNetwork/topia/api/web3/types"
)

//具体的api调用过程
//gasPrice没有专门的api，因此选用最新5个区块平均获得
func GasPriceHandler(parmas interface{}, apiServant interface{}) interface{} {
	//调用方法
	gasPrice, _ := types.EstimateTipFee()
	//3.进行类型转换，将该api的返回值包装为JsonRpcMessage
	enc, err := json.Marshal(gasPrice)
	if err != nil {
		//TODO:
		return nil //msg.ErrorResponse(err)
	}
	return &types.JsonrpcMessage{Result: enc}
}
