package universal

import (
	"context"
	"encoding/json"
	"fmt"
	"github.com/TopiaNetwork/topia/codec"
	tpcrtypes "github.com/TopiaNetwork/topia/crypt/types"
	tplog "github.com/TopiaNetwork/topia/log"
	txaction "github.com/TopiaNetwork/topia/transaction/action"
	txbasic "github.com/TopiaNetwork/topia/transaction/basic"
	"math/big"
)

type TransactionUniversalType uint32

const (
	TransactionUniversalType_Unknown TransactionUniversalType = iota
	TransactionUniversalType_Transfer
	TransactionUniversalType_ContractDeploy
	TransactionUniversalType_ContractInvoke
	TransactionUniversalType_NativeInvoke
	TransactionUniversalType_Relay
	TransactionUniversalType_DataTransfer
)

type TransactionUniversalWithHead struct {
	txbasic.TransactionHead
	TransactionUniversal
}

func NewTransactionUniversalWithHead(txHead *txbasic.TransactionHead, txUni *TransactionUniversal) *TransactionUniversalWithHead {
	return &TransactionUniversalWithHead{
		TransactionHead:      *txHead,
		TransactionUniversal: *txUni,
	}
}

func ContructTransactionUniversalWithHead(txHead *txbasic.TransactionHead, txUniHead *TransactionUniversalHead, txUnidata []byte) *TransactionUniversalWithHead {
	txUni := TransactionUniversal{
		Head: txUniHead,
		Data: &TransactionUniversalData{
			Specification: txUnidata,
		},
	}

	return &TransactionUniversalWithHead{
		TransactionHead:      *txHead,
		TransactionUniversal: txUni,
	}
}

func (txuni *TransactionUniversalWithHead) DataLen() uint64 {
	return uint64(len(txuni.Data.Specification))
}

func (txuni *TransactionUniversalWithHead) TxUniVerify(ctx context.Context, log tplog.Logger, nodeID string, txServant txbasic.TransactionServant) txbasic.VerifyResult {
	txUniServant := NewTransactionUniversalServant(txServant)

	marshaler := codec.CreateMarshaler(codec.CodecType_PROTO)
	txUniBytes, err := marshaler.Marshal(&txuni.TransactionUniversal)
	if err != nil {
		return txbasic.VerifyResult_Reject
	}

	tx := &txbasic.Transaction{
		Head: &txuni.TransactionHead,
		Data: &txbasic.TransactionData{
			Specification: txUniBytes,
		},
	}

	vR := tx.BasicVerify(ctx, log, txServant)
	switch vR {
	case txbasic.VerifyResult_Reject:
		return txbasic.VerifyResult_Reject
	case txbasic.VerifyResult_Ignore:
	case txbasic.VerifyResult_Accept:
		return ApplyTransactionUniversalVerifiers(ctx, log, nodeID, txuni, txUniServant,
			TransactionUniversalPayerAddressVerifier(),
			TransactionUniversalGasVerifier(),
			TransactionUniversalNonceVerifier(),
			TransactionUniversalPayerSignatureVerifier(),
		)
	default:
		panic("Invalid verify result")
	}

	return txbasic.VerifyResult_Accept
}

func (txuni *TransactionUniversalWithHead) Estimate(ctx context.Context, log tplog.Logger, nodeID string, txServant txbasic.TransactionServant) (*big.Int, error) {
	txUniServant := NewTransactionUniversalServant(txServant)

	gasEstimator, _ := txUniServant.GetGasEstimator()

	return gasEstimator.Estimate(ctx, log, nodeID, txServant, txuni)
}

func (m *TransactionUniversal) GetSpecificTransactionAction(txHead *txbasic.TransactionHead) txaction.TransactionAction {
	switch TransactionUniversalType(m.Head.Type) {
	case TransactionUniversalType_Transfer:
		var txUniTrData struct {
			TargetAddr tpcrtypes.Address
			Targets    []TargetItem
		}
		err := json.Unmarshal(m.Data.Specification, &txUniTrData)
		if err != nil {
			panic("Unmarshal tx uni data err: " + err.Error())
		}

		return NewTransactionUniversalTransfer(txHead, m.Head, txUniTrData.TargetAddr, txUniTrData.Targets)
	case TransactionUniversalType_ContractDeploy:
		var txUniDPData struct {
			ContractAddress tpcrtypes.Address
			Code            []byte
		}
		err := json.Unmarshal(m.Data.Specification, &txUniDPData)
		if err != nil {
			panic("Unmarshal tx uni data err: " + err.Error())
		}
		return NewTransactionUniversalDeploy(txHead, m.Head, txUniDPData.ContractAddress, txUniDPData.Code)
	case TransactionUniversalType_ContractInvoke:
		var txUniIVData struct {
			ContractAddr tpcrtypes.Address
			Method       string
			Args         string
		}
		err := json.Unmarshal(m.Data.Specification, &txUniIVData)
		if err != nil {
			panic("Unmarshal tx uni data err: " + err.Error())
		}
		return NewTransactionUniversalInvoke(txHead, m.Head, txUniIVData.ContractAddr, txUniIVData.Method, txUniIVData.Args)
	default:
		panic("Invalid tx uni type: " + fmt.Sprintf("%s", TransactionUniversalType(m.Head.Type).String()))
	}

	return nil
}

func (tr TransactionUniversalType) String() string {
	switch tr {
	case TransactionUniversalType_Transfer:
		return "Transfer"
	case TransactionUniversalType_ContractDeploy:
		return "ContractDeploy"
	case TransactionUniversalType_ContractInvoke:
		return "ContractInvoke"
	case TransactionUniversalType_NativeInvoke:
		return "NativeInvoke"
	case TransactionUniversalType_Relay:
		return "Relay"
	case TransactionUniversalType_DataTransfer:
		return "DataTransfer"
	default:
		return "Unknown"
	}
}

func (tr TransactionUniversalType) Value(trs string) TransactionUniversalType {
	switch trs {
	case "Transfer":
		return TransactionUniversalType_Transfer
	case "ContractDeploy":
		return TransactionUniversalType_ContractDeploy
	case "ContractInvoke":
		return TransactionUniversalType_ContractInvoke
	case "NativeInvoke":
		return TransactionUniversalType_NativeInvoke
	case "Relay":
		return TransactionUniversalType_Relay
	case "DataTransfer":
		return TransactionUniversalType_DataTransfer
	default:
		return TransactionUniversalType_Unknown
	}
}
