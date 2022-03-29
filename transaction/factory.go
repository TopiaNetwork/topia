package transaction

import (
	"github.com/TopiaNetwork/topia/codec"
	tpcrtypes "github.com/TopiaNetwork/topia/crypt/types"
	tplog "github.com/TopiaNetwork/topia/log"
	txaction "github.com/TopiaNetwork/topia/transaction/action"
	txbasic "github.com/TopiaNetwork/topia/transaction/basic"
	"github.com/TopiaNetwork/topia/transaction/universal"
)

func CreatTransaction(log tplog.Logger, privKey tpcrtypes.PrivateKey, txFromAddr tpcrtypes.Address, txCategory txbasic.TransactionCategory, txVersion txbasic.TransactionVersion, data []byte) *txbasic.Transaction {
	return txbasic.NewTransaction(log, privKey, txFromAddr, txCategory, txVersion, data)
}

func CreatTransactionAction(tx *txbasic.Transaction) txaction.TransactionAction {
	if tx == nil || tx.Head == nil || tx.Data == nil {
		panic("Invlaid tx input and can't create tx action")
	}

	marshaler := codec.CreateMarshaler(codec.CodecType_PROTO)

	switch txbasic.TransactionCategory(tx.Head.Category) {
	case txbasic.TransactionCategory_Topia_Universal:
		var txUni universal.TransactionUniversal
		err := marshaler.Unmarshal(tx.Data.Specification, &txUni)
		if err != nil {
			panic("Unmarshal tx data: " + err.Error())
		}

		return txUni.GetSpecificTransactionAction(tx.Head)
	default:
		panic("Invalid tx Category")
	}

	return nil
}
