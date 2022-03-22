package transaction

import (
	"context"
	"github.com/TopiaNetwork/topia/chain"
	tplog "github.com/TopiaNetwork/topia/log"
	"math/big"
)

const (
	TRANSFER_TargetItemMaxSize = 50
)

type TargetItem struct {
	Symbol chain.TokenSymbol
	Value  *big.Int
}

type TransactionTransfer struct {
	*Transaction
	Target []TargetItem
}

func (txfer *TransactionTransfer) Verify(ctx context.Context, log tplog.Logger, txServant TansactionServant) VerifyResult {
	vrResult := txfer.BasicVerify(ctx, log, txServant)
	if vrResult == VerifyResult_Reject {
		log.Error("Tx basic verify failed")
		return vrResult
	}

	targetItemSize := uint64(len(txfer.Target))
	if targetItemSize > txServant.GetChainConfig().MaxTargetItem {
		log.Errorf("Transfer target item size %d reaches max size", targetItemSize, txServant.GetChainConfig().MaxTargetItem)
		return VerifyResult_Reject
	}

	return VerifyResult_Accept
}

func (txfer *TransactionTransfer) Execute(ctx context.Context, log tplog.Logger, txServant TansactionServant) *TransactionResult {

	txHashBytes, _ := txfer.HashBytes()

	return &TransactionResult{
		TxHash:    txHashBytes,
		GasUsed:   0,
		ErrString: []byte(""),
		Status:    TransactionResult_OK,
	}
}
