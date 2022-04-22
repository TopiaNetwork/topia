package handlers

import (
	"encoding/json"
	"github.com/TopiaNetwork/topia/api/servant"
	"github.com/TopiaNetwork/topia/api/web3/types"
	tpcrtypes "github.com/TopiaNetwork/topia/crypt/types"
	"strconv"
)

//具体的api调用过程
func GetCodeHandler(parmas interface{}, apiServant interface{}) interface{} {
	//获取参数
	//TODO：参数格式校验
	args := parmas.(*types.GetCodeRequestType)
	servant := apiServant.(servant.APIServant)
	//调用方法
	height, _ := strconv.ParseUint(args.Height, 10, 64)
	code, err := servant.GetContractCode(tpcrtypes.NewFromString(args.Address), height)
	//3.进行类型转换，将该api的返回值包装为JsonRpcMessage
	if err != nil {
		errCode := types.ErrorResponse(err)
		return errCode
	}
	enc, err := json.Marshal(code)
	if err != nil {
		//TODO:
		return nil //msg.ErrorResponse(err)
	}
	return &types.JsonrpcMessage{Result: enc}
}
