package handlers

import (
	"encoding/json"
	"github.com/TopiaNetwork/topia/api/servant"
	"github.com/TopiaNetwork/topia/api/web3/types"
)

//具体的api调用过程
func GetBlockNumberHandler(parmas interface{}, apiServant interface{}) interface{} {
	//获取参数
	servant := apiServant.(servant.APIServant)
	//调用方法
	block, err := servant.GetLatestBlock()
	height := block.GetHead().GetHeight()
	//3.进行类型转换，将该api的返回值包装为JsonRpcMessage
	enc, err := json.Marshal(height)
	if err != nil {
		//TODO:
		return nil //msg.ErrorResponse(err)
	}
	return &types.JsonrpcMessage{Result: enc}
}
