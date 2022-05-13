package handlers

import (
	"encoding/json"
	"github.com/TopiaNetwork/topia/api/servant"
	"github.com/TopiaNetwork/topia/api/web3/types"
	types2 "github.com/TopiaNetwork/topia/api/web3/types/hexutil"
	tpcrtypes "github.com/TopiaNetwork/topia/crypt/types"
	"github.com/TopiaNetwork/topia/currency"
	"math/big"
)

func GetBalanceHandler(parmas interface{}, apiServant interface{}) interface{} {
	args := parmas.(*types.GetBalanceRequestType)
	servant := apiServant.(servant.APIServant)

	balance, err := servant.GetBalance(currency.TokenSymbol_Native, tpcrtypes.NewFromString(args.Address))
	if err != nil {
		return types.ErrorMessage(err)
	}
	var enc []byte
	if balance.BitLen() != 0 {
		enc, err = json.Marshal((*types2.Big)(balance))
	} else {
		enc, err = json.Marshal((*types2.Big)(big.NewInt(0)))
	}
	if err != nil {
		return types.ErrorMessage(err)
	}
	return &types.JsonrpcMessage{Result: enc}
}
