package handlers

import (
	"context"
	"encoding/hex"
	"encoding/json"
	"github.com/TopiaNetwork/topia/api/service"
	"github.com/TopiaNetwork/topia/api/web3/types"
)

func SendRawTransactionHandler(parmas interface{}, apiServant interface{}) interface{} {
	txService := &service.Transaction{}
	//获取参数
	args := parmas.(*types.SendRawTransactionRequestType)
	//调用方法
	tx, _ := types.ConstructTopiaTransaction(*args)
	ctx := context.Background()
	err := txService.SendTransaction(ctx, tx)
	//hash, _ := tx.HashBytes()
	//3.进行类型转换，将该api的返回值包装为JsonRpcMessage
	//TODO:需要返回txHash，topia格式的
	from := hex.EncodeToString(tx.GetHead().GetFromAddr())
	enc, err := json.Marshal(from)
	if err != nil {
		//TODO:
		return nil //msg.ErrorResponse(err)
	}
	return &types.JsonrpcMessage{Result: enc}
}
