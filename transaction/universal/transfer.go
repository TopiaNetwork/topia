package universal

import (
	"context"
	"encoding/json"
	"fmt"
	"math/big"

	tpcmm "github.com/TopiaNetwork/topia/common"
	"github.com/TopiaNetwork/topia/currency"

	"github.com/TopiaNetwork/topia/codec"
	tpcrtypes "github.com/TopiaNetwork/topia/crypt/types"
	tplog "github.com/TopiaNetwork/topia/log"
	txbasic "github.com/TopiaNetwork/topia/transaction/basic"
)

const (
	TRANSFER_TargetItemMaxSize = 50
)

type TargetItem struct {
	Symbol currency.TokenSymbol
	Value  *big.Int
}

type TransactionUniversalTransfer struct {
	txbasic.TransactionHead
	TransactionUniversalHead
	TargetAddr tpcrtypes.Address
	Targets    []TargetItem
}

func NewTransactionUniversalTransfer(txHead *txbasic.TransactionHead, txUniHead *TransactionUniversalHead, targetAddr tpcrtypes.Address, targets []TargetItem) *TransactionUniversalTransfer {
	return &TransactionUniversalTransfer{
		TransactionHead:          *txHead,
		TransactionUniversalHead: *txUniHead,
		TargetAddr:               targetAddr,
		Targets:                  targets,
	}
}

func (txfer *TransactionUniversalTransfer) DataBytes() ([]byte, error) {
	return json.Marshal(&struct {
		TargetAddr tpcrtypes.Address
		Targets    []TargetItem
	}{
		txfer.TargetAddr,
		txfer.Targets,
	})
}

func (txfer *TransactionUniversalTransfer) HashBytes() ([]byte, error) {
	marshaler := codec.CreateMarshaler(codec.CodecType_PROTO)

	txferData, _ := txfer.DataBytes()
	txUni := TransactionUniversal{
		Head: &txfer.TransactionUniversalHead,
		Data: &TransactionUniversalData{
			Specification: txferData,
		},
	}
	txUniBytes, err := marshaler.Marshal(&txUni)
	if err != nil {
		return nil, err
	}

	tx := &txbasic.Transaction{
		Head: &txfer.TransactionHead,
		Data: &txbasic.TransactionData{
			Specification: txUniBytes,
		},
	}

	return tx.HashBytes()
}

func (txfer *TransactionUniversalTransfer) Verify(ctx context.Context, log tplog.Logger, nodeID string, txServant txbasic.TransactionServant) txbasic.VerifyResult {
	txUniServant := NewTransactionUniversalServant(txServant)
	txUniData, _ := txfer.DataBytes()
	txUni := TransactionUniversal{
		Head: &txfer.TransactionUniversalHead,
		Data: &TransactionUniversalData{
			Specification: txUniData,
		},
	}

	txUniWithHead := &TransactionUniversalWithHead{
		TransactionHead:      txfer.TransactionHead,
		TransactionUniversal: txUni,
	}

	vR := txUniWithHead.TxUniVerify(ctx, log, nodeID, txServant)
	switch vR {
	case txbasic.VerifyResult_Reject:
		return txbasic.VerifyResult_Reject
	case txbasic.VerifyResult_Ignore:
	case txbasic.VerifyResult_Accept:
		return ApplyTransactionUniversalTransferVerifiers(ctx, log, txfer, txUniServant,
			TransactionUniversalTransferTargetAddressVerifier(),
			TransactionUniversalTransferTargetItemsVerifier(),
		)
	default:
		panic("Invalid verify result")
	}

	return txbasic.VerifyResult_Accept
}

func (txfer *TransactionUniversalTransfer) currencyTransfer(txServant txbasic.TransactionServant) (gasUsed uint64, errMsg string, status TransactionResultUniversal_ResultStatus) {
	dataBytes, _ := txfer.DataBytes()
	txUniWithHead := &TransactionUniversalWithHead{
		TransactionHead: txfer.TransactionHead,
		TransactionUniversal: TransactionUniversal{
			Head: &txfer.TransactionUniversalHead,
			Data: &TransactionUniversalData{
				Specification: dataBytes,
			},
		},
	}
	gasUsed = computeBasicGas(txServant.GetGasConfig(), txUniWithHead)
	gasVal := tpcmm.SafeMul(gasUsed, txfer.GasPrice)

	fromAcc, err := txServant.GetAccount(tpcrtypes.NewFromBytes(txfer.FromAddr))
	if err != nil {
		errMsg = err.Error()
		status = TransactionResultUniversal_Err
		return
	}
	targetAcc, err := txServant.GetAccount(txfer.TargetAddr)
	if err != nil {
		errMsg = err.Error()
		status = TransactionResultUniversal_Err
		return
	}
	feePayerAcc, err := txServant.GetAccount(tpcrtypes.NewFromBytes(txfer.FeePayer))
	if err != nil {
		errMsg = err.Error()
		status = TransactionResultUniversal_Err
		return
	}

	feePayerBal := feePayerAcc.Balances[currency.TokenSymbol_Native]
	if feePayerBal.Cmp(gasVal) < 0 {
		errMsg = fmt.Sprintf("Insufficient currency for gas fee: fee payer %s, remaining %s, required %s", txfer.FeePayer, feePayerBal.String(), gasVal.String())
		status = TransactionResultUniversal_Err
		return
	}

	for _, targetItem := range txfer.Targets {
		if fromCurrency, ok := fromAcc.Balances[targetItem.Symbol]; !ok {
			errMsg = fmt.Sprintf("Insufficient currency to transfer: from addr %s, currency %s remaining 0, required %s", txfer.FromAddr, targetItem.Symbol, targetItem.Value.String())
			status = TransactionResultUniversal_Err
			return
		} else if fromCurrency.Cmp(targetItem.Value) < 0 {
			errMsg = fmt.Sprintf("Insufficient currency to transfer: from addr %s, currency %s remaining %s, required %s", txfer.FromAddr, targetItem.Symbol, fromCurrency.String(), targetItem.Value.String())
			status = TransactionResultUniversal_Err
			return
		}
	}

	for _, targetItem := range txfer.Targets {
		targetAcc.BalanceIncrease(targetItem.Symbol, targetItem.Value)
		fromAcc.BalanceDecrease(targetItem.Symbol, targetItem.Value)
	}

	feePayerBal.Sub(feePayerBal, gasVal)

	status = TransactionResultUniversal_OK

	return
}

func (txfer *TransactionUniversalTransfer) Execute(ctx context.Context, log tplog.Logger, nodeID string, txServant txbasic.TransactionServant) *txbasic.TransactionResult {
	gasUsed, errMsg, status := txfer.currencyTransfer(txServant)

	txHashBytes, _ := txfer.HashBytes()
	txUniRS := &TransactionResultUniversal{
		Version:   txfer.TransactionHead.Version,
		TxHash:    txHashBytes,
		GasUsed:   gasUsed,
		ErrString: []byte(errMsg),
		Status:    status,
	}

	marshaler := codec.CreateMarshaler(codec.CodecType_PROTO)
	txUniRSBytes, err := marshaler.Marshal(txUniRS)
	if err != nil {
		return nil
	}

	return &txbasic.TransactionResult{
		Head: &txbasic.TransactionResultHead{
			Category: txfer.TransactionHead.Category,
			Version:  txfer.TransactionHead.Version,
			ChainID:  txfer.TransactionHead.ChainID,
		},
		Data: &txbasic.TransactionResultData{
			Specification: txUniRSBytes,
		},
	}
}
