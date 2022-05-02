package universal

import (
	"context"
	"encoding/json"
	"github.com/TopiaNetwork/topia/codec"
	tpcrtypes "github.com/TopiaNetwork/topia/crypt/types"
	tplog "github.com/TopiaNetwork/topia/log"
	txbasic "github.com/TopiaNetwork/topia/transaction/basic"
)

type TransactionUniversalDeploy struct {
	txbasic.TransactionHead
	TransactionUniversalHead
	ContractAddress tpcrtypes.Address
	Code            []byte
}

func NewTransactionUniversalDeploy(txHead *txbasic.TransactionHead, txUniHead *TransactionUniversalHead, contractAddress tpcrtypes.Address, code []byte) *TransactionUniversalDeploy {
	return &TransactionUniversalDeploy{
		TransactionHead:          *txHead,
		TransactionUniversalHead: *txUniHead,
		ContractAddress:          contractAddress,
		Code:                     code,
	}
}

func (txdp *TransactionUniversalDeploy) DataBytes() ([]byte, error) {
	return json.Marshal(&struct {
		ContractAddress tpcrtypes.Address
		Code            []byte
	}{
		txdp.ContractAddress,
		txdp.Code,
	})
}

func (txdp *TransactionUniversalDeploy) HashBytes() ([]byte, error) {
	marshaler := codec.CreateMarshaler(codec.CodecType_PROTO)

	txDPData, _ := txdp.DataBytes()
	txUni := TransactionUniversal{
		Head: &txdp.TransactionUniversalHead,
		Data: &TransactionUniversalData{
			Specification: txDPData,
		},
	}
	txUniBytes, err := marshaler.Marshal(&txUni)
	if err != nil {
		return nil, err
	}

	tx := &txbasic.Transaction{
		Head: &txdp.TransactionHead,
		Data: &txbasic.TransactionData{
			Specification: txUniBytes,
		},
	}

	return tx.HashBytes()
}

func (txdp *TransactionUniversalDeploy) Verify(ctx context.Context, log tplog.Logger, txServant txbasic.TransactionServant) txbasic.VerifyResult {
	txUniServant := NewTransactionUniversalServant(txServant)
	txUniData, _ := txdp.DataBytes()
	txUni := TransactionUniversal{
		Head: &txdp.TransactionUniversalHead,
		Data: &TransactionUniversalData{
			Specification: txUniData,
		},
	}

	txUniWithHead := &TransactionUniversalWithHead{
		TransactionHead:      txdp.TransactionHead,
		TransactionUniversal: txUni,
	}

	vR := txUniWithHead.TxUniVerify(ctx, log, txServant)
	switch vR {
	case txbasic.VerifyResult_Reject:
		return txbasic.VerifyResult_Reject
	case txbasic.VerifyResult_Ignore:
	case txbasic.VerifyResult_Accept:
		return ApplyTransactionUniversalDeployVerifiers(ctx, log, txdp, txUniServant,
			TransactionUniversalDeployContractAddressVerifier(),
			TransactionUniversalDeployCodeVerifier(),
		)
	default:
		panic("Invalid verify result")
	}

	return txbasic.VerifyResult_Accept
}

func (txdp *TransactionUniversalDeploy) Execute(ctx context.Context, log tplog.Logger, txServant txbasic.TransactionServant) *txbasic.TransactionResult {
	txHashBytes, _ := txdp.HashBytes()
	txUniRS := &TransactionResultUniversal{
		Version:   txdp.TransactionHead.Version,
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
			Category: txdp.TransactionHead.Category,
			Version:  txdp.TransactionHead.Version,
			ChainID:  txdp.TransactionHead.ChainID,
		},
		Data: &txbasic.TransactionResultData{
			Specification: txUniRSBytes,
		},
	}
}
