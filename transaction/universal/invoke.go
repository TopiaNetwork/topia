package universal

import (
	"context"
	"encoding/json"
	"github.com/TopiaNetwork/topia/codec"
	tpcrtypes "github.com/TopiaNetwork/topia/crypt/types"
	tplog "github.com/TopiaNetwork/topia/log"
	txbasic "github.com/TopiaNetwork/topia/transaction/basic"
	tpvm "github.com/TopiaNetwork/topia/vm"
	tpvmmservice "github.com/TopiaNetwork/topia/vm/service"
	tpvmtype "github.com/TopiaNetwork/topia/vm/type"
	"math/big"
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

func (txiv *TransactionUniversalInvoke) DataBytes() ([]byte, error) {
	return json.Marshal(&struct {
		ContractAddr tpcrtypes.Address
		Method       string
		Args         string
	}{
		txiv.ContractAddr,
		txiv.Method,
		txiv.Args,
	})
}

func (txiv *TransactionUniversalInvoke) HashBytes() ([]byte, error) {
	marshaler := codec.CreateMarshaler(codec.CodecType_PROTO)

	txDPData, _ := txiv.DataBytes()
	txUni := TransactionUniversal{
		Head: &txiv.TransactionUniversalHead,
		Data: &TransactionUniversalData{
			Specification: txDPData,
		},
	}
	txUniBytes, err := marshaler.Marshal(&txUni)
	if err != nil {
		return nil, err
	}

	tx := &txbasic.Transaction{
		Head: &txiv.TransactionHead,
		Data: &txbasic.TransactionData{
			Specification: txUniBytes,
		},
	}

	return tx.HashBytes()
}

func (txiv *TransactionUniversalInvoke) Verify(ctx context.Context, log tplog.Logger, nodeID string, txServant txbasic.TransactionServant) txbasic.VerifyResult {
	txUniServant := NewTransactionUniversalServant(txServant)

	txUniData, _ := txiv.DataBytes()
	txUniWithHead := ContructTransactionUniversalWithHead(&txiv.TransactionHead, &txiv.TransactionUniversalHead, txUniData)

	vR := txUniWithHead.TxUniVerify(ctx, log, nodeID, txServant)
	switch vR {
	case txbasic.VerifyResult_Reject:
		return txbasic.VerifyResult_Reject
	case txbasic.VerifyResult_Ignore:
	case txbasic.VerifyResult_Accept:
		return ApplyTransactionUniversalInvokeVerifiers(ctx, log, txiv, txUniServant,
			TransactionUniversalInvokeContractAddressVerifier(),
			TransactionUniversalInvokeMethodVerifier(),
			TransactionUniversalInvokeArgsVerifier(),
		)
	default:
		panic("Invalid verify result")
	}

	return txbasic.VerifyResult_Accept
}

func (txiv *TransactionUniversalInvoke) Estimate(ctx context.Context, log tplog.Logger, nodeID string, txServant txbasic.TransactionServant) (*big.Int, error) {
	txUniData, _ := txiv.DataBytes()
	txUniWithHead := ContructTransactionUniversalWithHead(&txiv.TransactionHead, &txiv.TransactionUniversalHead, txUniData)

	return txUniWithHead.Estimate(ctx, log, nodeID, txServant)
}

func (txiv *TransactionUniversalInvoke) Execute(ctx context.Context, log tplog.Logger, nodeID string, txServant txbasic.TransactionServant) *txbasic.TransactionResult {
	vmServant := tpvmmservice.NewVMServant(txServant, txServant.GetGasConfig().MaxGasEachBlock)
	vmContext := &tpvmmservice.VMContext{
		Context:      ctx,
		VMServant:    vmServant,
		NodeID:       nodeID,
		ContractAddr: txiv.ContractAddr,
		Method:       txiv.Method,
		Args:         txiv.Args,
	}

	txUniData, _ := txiv.DataBytes()
	gasUsed := computeBasicGas(txServant.GetGasConfig(), uint64(len(txUniData)))

	errMsg := ""
	status := TransactionResultUniversal_Err

	vmType := tpvmtype.VMType_TVM
	if tpcrtypes.IsNativeContractAddress(txiv.ContractAddr) {
		vmType = tpvmtype.VMType_NATIVE
	}
	vmResult, err := tpvm.GetVMFactory().GetVM(vmType).ExecuteContract(vmContext)
	if err != nil {
		errMsg = err.Error()
	}
	if vmResult.Code != tpvmtype.ReturnCode_Ok {
		errMsg = vmResult.Code.String() + ": " + vmResult.ErrMsg
	}

	status = TransactionResultUniversal_OK
	gasUsed += vmResult.GasUsed

	txHashBytes, _ := txiv.HashBytes()
	txUniRS := &TransactionResultUniversal{
		Version:   txiv.TransactionHead.Version,
		TxHash:    txHashBytes,
		GasUsed:   gasUsed,
		ErrString: []byte(errMsg),
		Status:    status,
		Data:      vmResult.Data,
	}

	marshaler := codec.CreateMarshaler(codec.CodecType_PROTO)
	txUniRSBytes, err := marshaler.Marshal(txUniRS)
	if err != nil {
		return nil
	}

	return &txbasic.TransactionResult{
		Head: &txbasic.TransactionResultHead{
			Category: txiv.TransactionHead.Category,
			Version:  txiv.TransactionHead.Version,
			ChainID:  txiv.TransactionHead.ChainID,
		},
		Data: &txbasic.TransactionResultData{
			Specification: txUniRSBytes,
		},
	}
}
