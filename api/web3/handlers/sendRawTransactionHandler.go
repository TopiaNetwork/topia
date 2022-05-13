package handlers

import (
	"context"
	"encoding/json"
	"github.com/TopiaNetwork/topia/api/web3/types"
)

func SendRawTransactionHandler(parmas interface{}, apiServant interface{}) interface{} {
	args := parmas.(*types.SendRawTransactionRequestType)
	txIn := apiServant.(types.TxInterface)
	tx, _ := types.ConstructTopiaTransaction(*args)
	ctx := context.Background()
	err := txIn.SendTransaction(ctx, tx)
	if err != nil {
		return types.ErrorMessage(err)
	}
	hashByte, _ := tx.HashBytes()
	var hash types.Hash
	hash.SetBytes(hashByte)

	enc, err := json.Marshal(hash)
	if err != nil {
		return types.ErrorMessage(err)
	}
	return &types.JsonrpcMessage{Result: enc}
}
