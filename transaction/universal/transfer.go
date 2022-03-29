package universal

import (
	"context"
	"encoding/json"
	"github.com/TopiaNetwork/topia/codec"
	"math/big"

	"github.com/TopiaNetwork/topia/chain"
	tpcrtypes "github.com/TopiaNetwork/topia/crypt/types"
	tplog "github.com/TopiaNetwork/topia/log"
	txbasic "github.com/TopiaNetwork/topia/transaction/basic"
)

const (
	TRANSFER_TargetItemMaxSize = 50
)

type TargetItem struct {
	Symbol chain.TokenSymbol
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

func (txfer *TransactionUniversalTransfer) Verify(ctx context.Context, log tplog.Logger, txServant txbasic.TansactionServant) txbasic.VerifyResult {
	txUniServant := NewTansactionUniversalServant(txServant)
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

	vR := txUniWithHead.TxUniVerify(ctx, log, txServant)
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

func (txfer *TransactionUniversalTransfer) Execute(ctx context.Context, log tplog.Logger, txServant txbasic.TansactionServant) *txbasic.TransactionResult {
	txHashBytes, _ := txfer.HashBytes()
	txUniRS := &TransactionResultUniversal{
		Version:   txfer.TransactionHead.Version,
		TxHash:    txHashBytes,
		GasUsed:   0,
		ErrString: []byte(""),
		Status:    TransactionResultUniversal_OK,
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
