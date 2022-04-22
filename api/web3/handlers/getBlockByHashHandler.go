package handlers

import (
	"encoding/json"
	"github.com/TopiaNetwork/topia/api/servant"
	"github.com/TopiaNetwork/topia/api/web3/types"
)

//具体的api调用过程
func GetBlockByHashHandler(parmas interface{}, apiServant interface{}) interface{} {
	//获取参数
	//TODO：参数格式校验
	args := parmas.(*types.GetBlockByHashRequestType)
	servant := apiServant.(servant.APIServant)
	//调用方法
	block, err := servant.GetBlockByHash(args.BlockHash)
	//3.进行类型转换，将该api的返回值包装为JsonRpcMessage
	ethBlock := types.ConstructGetBlockResponseType(block)
	enc, err := json.Marshal(ethBlock)
	if err != nil {
		//TODO:
		return nil //msg.ErrorResponse(err)
	}
	return &types.JsonrpcMessage{Result: enc}
}
