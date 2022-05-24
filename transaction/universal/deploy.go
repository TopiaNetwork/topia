package universal

import (
	"context"
	"encoding/json"
	"math/big"

	"github.com/TopiaNetwork/topia/codec"
	tpcrtypes "github.com/TopiaNetwork/topia/crypt/types"
	tplog "github.com/TopiaNetwork/topia/log"
	txbasic "github.com/TopiaNetwork/topia/transaction/basic"
	tpvm "github.com/TopiaNetwork/topia/vm"
	tpvmmservice "github.com/TopiaNetwork/topia/vm/service"
	tpvmtype "github.com/TopiaNetwork/topia/vm/type"
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

func (txdp *TransactionUniversalDeploy) Verify(ctx context.Context, log tplog.Logger, nodeID string, txServant txbasic.TransactionServant) txbasic.VerifyResult {
	txUniServant := NewTransactionUniversalServant(txServant)

	txUniData, _ := txdp.DataBytes()
	txUniWithHead := ContructTransactionUniversalWithHead(&txdp.TransactionHead, &txdp.TransactionUniversalHead, txUniData)

	vR := txUniWithHead.TxUniVerify(ctx, log, nodeID, txServant)
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

func (txdp *TransactionUniversalDeploy) Estimate(ctx context.Context, log tplog.Logger, nodeID string, txServant txbasic.TransactionServant) (*big.Int, error) {
	txUniData, _ := txdp.DataBytes()
	txUniWithHead := ContructTransactionUniversalWithHead(&txdp.TransactionHead, &txdp.TransactionUniversalHead, txUniData)

	return txUniWithHead.Estimate(ctx, log, nodeID, txServant)
}

func (txdp *TransactionUniversalDeploy) Execute(ctx context.Context, log tplog.Logger, nodeID string, txServant txbasic.TransactionServant) *txbasic.TransactionResult {
	vmServant := tpvmmservice.NewVMServant(txServant, txServant.GetGasConfig().MaxGasEachBlock)
	vmContext := &tpvmmservice.VMContext{
		Context:   ctx,
		VMServant: vmServant,
		NodeID:    nodeID,
		Code:      txdp.Code,
	}

	txUniData, _ := txdp.DataBytes()
	gasUsed := computeBasicGas(txServant.GetGasConfig(), uint64(len(txUniData)))

	errMsg := ""
	status := TransactionResultUniversal_Err
	vmResult, err := tpvm.GetVMFactory().GetVM(tpvmtype.VMType_TVM).DeployContract(vmContext)
	if err != nil {
		errMsg = err.Error()
	}
	if vmResult.Code != tpvmtype.ReturnCode_Ok {
		errMsg = vmResult.Code.String() + ": " + vmResult.ErrMsg
	}

	status = TransactionResultUniversal_OK

	txHashBytes, _ := txdp.HashBytes()
	txUniRS := &TransactionResultUniversal{
		Version:   txdp.TransactionHead.Version,
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
			Category: txdp.TransactionHead.Category,
			Version:  txdp.TransactionHead.Version,
			ChainID:  txdp.TransactionHead.ChainID,
		},
		Data: &txbasic.TransactionResultData{
			Specification: txUniRSBytes,
		},
	}
}
