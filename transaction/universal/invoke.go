package universal

import (
	"context"
	"encoding/json"
	"github.com/TopiaNetwork/topia/codec"
	tpcrtypes "github.com/TopiaNetwork/topia/crypt/types"
	tplog "github.com/TopiaNetwork/topia/log"
	txbasic "github.com/TopiaNetwork/topia/transaction/basic"
	tpvm "github.com/TopiaNetwork/topia/vm"
	tpvmcmm "github.com/TopiaNetwork/topia/vm/common"
)

type TransactionUniversalInvoke struct {
	txbasic.TransactionHead
	TransactionUniversalHead
	ContractAddr tpcrtypes.Address
	Method       string
	Args         string
}

func NewTransactionUniversalInvoke(txHead *txbasic.TransactionHead, txUniHead *TransactionUniversalHead, contractAddr tpcrtypes.Address, method string, args string) *TransactionUniversalInvoke {
	return &TransactionUniversalInvoke{
		TransactionHead:          *txHead,
		TransactionUniversalHead: *txUniHead,
		ContractAddr:             contractAddr,
		Method:                   method,
		Args:                     args,
	}
}

func (txIV *TransactionUniversalInvoke) DataBytes() ([]byte, error) {
	return json.Marshal(&struct {
		ContractAddr tpcrtypes.Address
		Method       string
		Args         string
	}{
		txIV.ContractAddr,
		txIV.Method,
		txIV.Args,
	})
}

func (txIV *TransactionUniversalInvoke) HashBytes() ([]byte, error) {
	marshaler := codec.CreateMarshaler(codec.CodecType_PROTO)

	txDPData, _ := txIV.DataBytes()
	txUni := TransactionUniversal{
		Head: &txIV.TransactionUniversalHead,
		Data: &TransactionUniversalData{
			Specification: txDPData,
		},
	}
	txUniBytes, err := marshaler.Marshal(&txUni)
	if err != nil {
		return nil, err
	}

	tx := &txbasic.Transaction{
		Head: &txIV.TransactionHead,
		Data: &txbasic.TransactionData{
			Specification: txUniBytes,
		},
	}

	return tx.HashBytes()
}

func (txIV *TransactionUniversalInvoke) Verify(ctx context.Context, log tplog.Logger, nodeID string, txServant txbasic.TransactionServant) txbasic.VerifyResult {
	txUniServant := NewTransactionUniversalServant(txServant)
	txUniData, _ := txIV.DataBytes()
	txUni := TransactionUniversal{
		Head: &txIV.TransactionUniversalHead,
		Data: &TransactionUniversalData{
			Specification: txUniData,
		},
	}

	txUniWithHead := &TransactionUniversalWithHead{
		TransactionHead:      txIV.TransactionHead,
		TransactionUniversal: txUni,
	}

	vR := txUniWithHead.TxUniVerify(ctx, log, nodeID, txServant)
	switch vR {
	case txbasic.VerifyResult_Reject:
		return txbasic.VerifyResult_Reject
	case txbasic.VerifyResult_Ignore:
	case txbasic.VerifyResult_Accept:
		return ApplyTransactionUniversalInvokeVerifiers(ctx, log, txIV, txUniServant,
			TransactionUniversalInvokeContractAddressVerifier(),
			TransactionUniversalInvokeMethodVerifier(),
			TransactionUniversalInvokeArgsVerifier(),
		)
	default:
		panic("Invalid verify result")
	}

	return txbasic.VerifyResult_Accept
}

func (txIV *TransactionUniversalInvoke) Execute(ctx context.Context, log tplog.Logger, nodeID string, txServant txbasic.TransactionServant) *txbasic.TransactionResult {
	vmServant := tpvmcmm.NewVMServant(txServant, txServant.GetGasConfig().MaxGasEachBlock)
	vmContext := &tpvmcmm.VMContext{
		Context:      ctx,
		VMServant:    vmServant,
		NodeID:       nodeID,
		ContractAddr: txIV.ContractAddr,
		Method:       txIV.Method,
		Args:         txIV.Args,
	}

	gasUsed := uint64(0)
	errMsg := ""
	status := TransactionResultUniversal_Err
	vmResult, err := tpvm.GetVMFactory().GetVM(tpvmcmm.VMType_TVM).DeployContract(vmContext)
	if err != nil {
		errMsg = err.Error()
	}
	if vmResult.Code != tpvmcmm.ReturnCode_Ok {
		errMsg = vmResult.ErrMsg
	}

	status = TransactionResultUniversal_OK
	gasUsed = vmResult.GasUsed

	txHashBytes, _ := txIV.HashBytes()
	txUniRS := &TransactionResultUniversal{
		Version:   txIV.TransactionHead.Version,
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
			Category: txIV.TransactionHead.Category,
			Version:  txIV.TransactionHead.Version,
			ChainID:  txIV.TransactionHead.ChainID,
		},
		Data: &txbasic.TransactionResultData{
			Specification: txUniRSBytes,
		},
	}
}
