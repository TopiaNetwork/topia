package handlers

import (
	"encoding/json"
	"github.com/TopiaNetwork/topia/api/servant"
	"github.com/TopiaNetwork/topia/api/web3/types"
	tpcrtypes "github.com/TopiaNetwork/topia/crypt/types"
	"github.com/TopiaNetwork/topia/currency"
)

//具体的api调用过程
func GetBalanceHandler(parmas interface{}, apiServant interface{}) interface{} {
	//获取参数
	//TODO：参数格式校验
	args := parmas.(*types.GetBalanceRequestType)
	servant := apiServant.(servant.APIServant)
	//调用方法
	balance, err := servant.GetBalance(currency.TokenSymbol_Native, tpcrtypes.NewFromString(args.Address))
	//3.进行类型转换，将该api的返回值包装为JsonRpcMessage
	enc, err := json.Marshal(balance)
	if err != nil {
		//TODO:
		return nil //msg.ErrorResponse(err)
	}
	return &types.JsonrpcMessage{Result: enc}
}
