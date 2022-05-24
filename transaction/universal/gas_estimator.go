package universal

import (
	"context"
	"fmt"
	tplog "github.com/TopiaNetwork/topia/log"
	"math/big"

	tpcmm "github.com/TopiaNetwork/topia/common"
	txbasic "github.com/TopiaNetwork/topia/transaction/basic"
)

type GasEstimator interface {
	Estimate(ctx context.Context, log tplog.Logger, nodeID string, txServant txbasic.TransactionServant, txUni *TransactionUniversalWithHead) (*big.Int, error)
}

func NewGasEstimator() GasEstimator {
	return &gasEstimator{}
}

type gasEstimator struct {
}

func (ge *gasEstimator) Estimate(ctx context.Context, log tplog.Logger, nodeID string, txServant txbasic.TransactionServant, txUni *TransactionUniversalWithHead) (*big.Int, error) {
	gasUsed := computeBasicGas(txServant.GetGasConfig(), txUni.DataLen())
	gasPriceC, err := NewGasPriceComputer(txServant.GetMarshaler(), txServant.GetTxPoolSize, txServant.GetLatestBlock, txServant.GetGasConfig(), txServant.GetChainConfig()).ComputeGasPrice()
	if err != nil {
		log.Errorf("Can get latest gas price and estimate err: %v", err)
		return nil, nil
	}

	txUniGasPrice := txUni.Head.GasPrice
	if gasPriceC > txUni.Head.GasPrice {
		txUniGasPrice = gasPriceC
	}

	switch TransactionUniversalType(txUni.Head.Type) {
	case TransactionUniversalType_Transfer:
	case TransactionUniversalType_ContractDeploy:
		return tpcmm.SafeMul(gasUsed, txUniGasPrice), nil
	case TransactionUniversalType_NativeInvoke:
	case TransactionUniversalType_ContractInvoke:
		txRS := txUni.TransactionUniversal.GetSpecificTransactionAction(&txUni.TransactionHead).Execute(ctx, log, nodeID, txServant)
		txUniRS := TransactionResultUniversal{}
		err := txServant.GetMarshaler().Unmarshal(txRS.Data.Specification, &txUniRS)
		if err != nil {
			return nil, err
		}
		if txUniRS.Status == TransactionResultUniversal_OK {
			gasUsed, err = tpcmm.SafeAddUint64(gasUsed, txUniRS.GasUsed)
			if err != nil {
				return nil, err
			}
		} else {
			return nil, fmt.Errorf("%s", txUniRS.ErrString)
		}

		return tpcmm.SafeMul(gasUsed, txUniGasPrice), nil
	}

	return big.NewInt(0), nil
}
