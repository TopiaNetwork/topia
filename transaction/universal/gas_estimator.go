package universal

import (
	"math/big"

	tpcmm "github.com/TopiaNetwork/topia/common"
	txbasic "github.com/TopiaNetwork/topia/transaction/basic"
)

type GasEstimator interface {
	Estimate(txUni *TransactionUniversalWithHead) (*big.Int, error)
}

func NewGasEstimator(txServant txbasic.TransactionServant) GasEstimator {
	return &gasEstimator{}
}

type gasEstimator struct {
	txServant txbasic.TransactionServant
}

func (ge *gasEstimator) computeBasicGas(txUni *TransactionUniversalWithHead) uint64 {
	gasConfig := ge.txServant.GetGasConfig()

	return gasConfig.MinGasLimit + txUni.DataLen()*gasConfig.GasEachByte
}

func (ge *gasEstimator) Estimate(txUni *TransactionUniversalWithHead) (*big.Int, error) {
	switch TransactionUniversalType(txUni.Head.Type) {
	case TransactionUniversalType_Transfer:
	case TransactionUniversalType_ContractDeploy:
		gasUsed := ge.computeBasicGas(txUni)
		gasVal := tpcmm.SafeMul(gasUsed, txUni.Head.GasPrice)
		return gasVal, nil
	}
	return big.NewInt(0), nil
}
