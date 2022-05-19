package universal

import (
	"math"

	tpchaintypes "github.com/TopiaNetwork/topia/chain/types"
	"github.com/TopiaNetwork/topia/codec"
	tpconfig "github.com/TopiaNetwork/topia/configuration"
	txbasic "github.com/TopiaNetwork/topia/transaction/basic"
)

type GasPriceComputer interface {
	ComputeGasPrice() (uint64, error)
}

func NewGasPriceComputer(marshaler codec.Marshaler,
	txPoolSize func() int64,
	latestBlock func() (*tpchaintypes.Block, error),
	gasConfig *tpconfig.GasConfiguration,
	chainConfig *tpconfig.ChainConfiguration) GasPriceComputer {
	return &gasPriceComputer{
		txPoolSize:  txPoolSize,
		latestBlock: latestBlock,
		marshaler:   marshaler,
		gasConfig:   gasConfig,
		chainConfig: chainConfig,
	}
}

type gasPriceComputer struct {
	txPoolSize  func() int64
	latestBlock func() (*tpchaintypes.Block, error)
	marshaler   codec.Marshaler
	gasConfig   *tpconfig.GasConfiguration
	chainConfig *tpconfig.ChainConfiguration
}

func (gc *gasPriceComputer) getMinGasPriceOfLatestBlock() (uint64, error) {
	latestBlock, err := gc.latestBlock()
	if err != nil {
		return 0, err
	}

	if latestBlock.Head.TxCount == 0 {
		return gc.gasConfig.MinGasPrice, nil
	}

	var txMinGasPrice uint64
	for _, txBytes := range latestBlock.Data.Txs {
		var tx txbasic.Transaction
		err = gc.marshaler.Unmarshal(txBytes, &tx)
		if err != nil {
			panic("Unmarshal tx: " + err.Error())
		}

		switch txbasic.TransactionCategory(tx.Head.Category) {
		case txbasic.TransactionCategory_Topia_Universal:
			var txUni TransactionUniversal
			err = gc.marshaler.Unmarshal(tx.Data.Specification, &txUni)
			if err != nil {
				panic("Unmarshal tx data: " + err.Error())
			}

			if txMinGasPrice == 0 || txMinGasPrice > txUni.Head.GasPrice {
				txMinGasPrice = txUni.Head.GasPrice
			}
		}
	}

	if txMinGasPrice == 0 {
		txMinGasPrice = gc.gasConfig.MinGasPrice
	}

	return txMinGasPrice, nil
}

func (gc *gasPriceComputer) ComputeGasPrice() (uint64, error) {
	pendingBlock := uint64(0)
	txPoolSize := gc.txPoolSize()
	if uint64(txPoolSize)%gc.chainConfig.MaxTxSizeOfEachBlock > 0 {
		pendingBlock = uint64(txPoolSize)/gc.chainConfig.MaxTxSizeOfEachBlock + 1
	}

	if pendingBlock <= 1 { //idle
		return gc.gasConfig.MinGasPrice, nil
	}

	txMinGasPrice, err := gc.getMinGasPriceOfLatestBlock()
	if err != nil {
		return 0, err
	}

	tempGasPriceF := float64(txMinGasPrice)
	if pendingBlock > 1 {
		for i := uint64(0); i < pendingBlock-1; i++ {
			tempGasPriceF += math.Pow(tempGasPriceF, gc.gasConfig.GasPriceMultiple)
		}
	}

	tempGasPriceUInt64 := uint64(tempGasPriceF)

	if tempGasPriceUInt64 >= gc.gasConfig.MinGasPrice*5 {
		tempGasPriceUInt64 = gc.gasConfig.MinGasPrice * 5
	}

	return tempGasPriceUInt64, nil
}
