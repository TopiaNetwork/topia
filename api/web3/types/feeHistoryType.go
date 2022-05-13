package types

import (
	"encoding/json"
	"fmt"
	"github.com/TopiaNetwork/topia/api/servant"
	types "github.com/TopiaNetwork/topia/api/web3/types/hexutil"
	"github.com/TopiaNetwork/topia/codec"
	"github.com/TopiaNetwork/topia/transaction/universal"
	"math/big"
	"strconv"
)

type FeeHistoryRequestType struct {
	BlockCount  int
	BlockHeight string
	Percentile  []float64
}

type FeeHistoryResponseType struct {
	OldestBlock  *types.Big
	Reward       [][]*types.Big
	BaseFee      []*types.Big
	GasUsedRatio []float64
}

type (
	txGasAndReward struct {
		gasUsed uint64
		reward  uint64
	}
	sortGasAndReward []txGasAndReward
)

func Hex2Dec(val string) uint64 {
	n, err := strconv.ParseUint(val, 16, 32)
	if err != nil {
		fmt.Println(err)
	}
	return n
}

func FeeHistory(requestType *FeeHistoryRequestType, servant servant.APIServant) *FeeHistoryResponseType {
	fmt.Println(requestType.BlockHeight)
	lastBlock := Hex2Dec(requestType.BlockHeight[2:])
	blockCount := requestType.BlockCount
	percentiles := requestType.Percentile

	oldestBlock := lastBlock + 1 - uint64(blockCount)
	Reward := make([][]*big.Int, blockCount)
	BaseFee := make([]uint64, 0)
	GasUsedRatio := make([]float64, 0)

	gasLimit := 10_000_000_000

	for ; blockCount > 0; blockCount-- {
		BaseFee = append(BaseFee, 0)
		block, _ := servant.GetBlockByHeight(oldestBlock - uint64(blockCount))
		gasUsed := Hex2Dec(string(block.GetHead().GetGasFees())[2:])
		gasUsedRatio := float64(gasUsed) / float64(gasLimit)
		GasUsedRatio = append(GasUsedRatio, gasUsedRatio)
		sorter := make(sortGasAndReward, 0)

		txHashs := block.GetData().GetTxs()
		for _, txHash := range txHashs {
			transaction, _ := servant.GetTransactionByHash(string(txHash))
			var transactionUniversal universal.TransactionUniversal
			_ = json.Unmarshal(transaction.GetData().GetSpecification(), &transactionUniversal)
			transactionReceipt, _ := servant.GetTransactionResultByHash(string(txHash))
			var transactionResultUniversal universal.TransactionResultUniversal
			marshaler := codec.CreateMarshaler(codec.CodecType_PROTO)
			_ = marshaler.Unmarshal(transactionReceipt.GetData().GetSpecification(), &transactionResultUniversal)
			sorter = append(sorter, txGasAndReward{transactionResultUniversal.GetGasUsed(), transactionUniversal.GetHead().GetGasPrice()})
		}

		var txIndex int
		sumGasUsed := sorter[0].gasUsed
		reward := make([]*big.Int, len(percentiles))
		for i, p := range percentiles {
			thresholdGasUsed := float64(gasUsed) * p / 100
			for sumGasUsed < uint64(thresholdGasUsed) && txIndex < len(txHashs)-1 {
				txIndex++
				sumGasUsed += sorter[txIndex].gasUsed
			}
			reward[i] = new(big.Int).SetUint64(sorter[txIndex].reward)
		}
		Reward[blockCount-1] = reward
	}
	BaseFee = append(BaseFee, 0)
	rewardStr := make([][]*types.Big, len(Reward))
	baseFeeStr := make([]*types.Big, len(BaseFee))

	for i, v := range Reward {
		for iv, vv := range v {
			rewardStr[i] = make([]*types.Big, len(percentiles))
			rewardStr[i][iv] = (*types.Big)(vv)
		}
	}
	for i, _ := range BaseFee {
		baseFeeStr[i] = (*types.Big)(big.NewInt(0))
	}
	return &FeeHistoryResponseType{
		OldestBlock:  (*types.Big)(new(big.Int).SetUint64(oldestBlock)),
		Reward:       rewardStr,
		BaseFee:      baseFeeStr,
		GasUsedRatio: GasUsedRatio,
	}
}
