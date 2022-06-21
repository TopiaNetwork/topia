package handlers

import (
	"encoding/binary"
	"encoding/json"
	"fmt"
	"github.com/TopiaNetwork/topia/api/servant"
	hexutil "github.com/TopiaNetwork/topia/api/web3/eth/types/hexutil"
	"github.com/TopiaNetwork/topia/codec"
	txbasic "github.com/TopiaNetwork/topia/transaction/basic"
	txuni "github.com/TopiaNetwork/topia/transaction/universal"
	"math"
	"math/big"
	"math/rand"
	"sort"
	"strconv"
	"time"
)

var (
	gasLimit        = 10_000_000_000
	initBaseFee     = 1_000_000_000
	window          = 19
	checkBlocks     = 3
	ignorePrice     = 2
	sampleNumber    = 3
	defaultGasPrice = initBaseFee
	percentile      = 60
	maxPrice        = big.NewInt(500 * 1e9)
)

func estimateBaseFee() uint64 {
	randRange := []float64{0.5, 0.8, 1.0, 1.5}
	rand.Seed(time.Now().Unix())
	index := rand.Intn(4)
	randomBaseFee := randRange[index] * float64(initBaseFee)

	servant := servant.NewAPIServant()
	block, _ := servant.GetLatestBlock()
	height := block.GetHead().GetHeight()
	initHeight := height - uint64(window)
	for i := initHeight; i <= height; i++ {
		block, _ = servant.GetBlockByHeight(i)
		randomBaseFee = randomBaseFee * Float64frombytes(block.GetHead().GetGasFees()) / float64(gasLimit)
	}
	return uint64(randomBaseFee)
}
func Float64frombytes(bytes []byte) float64 {
	bits := binary.LittleEndian.Uint64(bytes)
	float := math.Float64frombits(bits)
	return float
}

func EstimateTipFee(apiServant servant.APIServant) (*big.Int, error) {
	block, _ := apiServant.GetLatestBlock()
	height := block.GetHead().GetHeight()

	var (
		sent, exp int
		number    = height
		result    = make(chan results, checkBlocks)
		quit      = make(chan struct{})
		results   []*big.Int
	)
	for sent < checkBlocks && number > 0 {
		go getBlockValues(number, sampleNumber, uint64(ignorePrice), result, quit, apiServant)
		sent++
		exp++
		number--
	}
	for exp > 0 {
		res := <-result
		if res.err != nil {
			close(quit)
			return big.NewInt(int64(defaultGasPrice)), res.err
		}
		exp--
		// Nothing returned. There are two special cases here:
		// - The block is empty
		// - All the transactions included are sent by the miner itself.
		// In these cases, use the latest calculated price for sampling.
		if len(res.values) == 0 {
			res.values = []*big.Int{big.NewInt(int64(defaultGasPrice))}
		}
		// Besides, in order to collect enough data for sampling, if nothing
		// meaningful returned, try to query more blocks. But the maximum
		// is 2*checkBlocks.
		if len(res.values) == 1 && len(results)+1+exp < checkBlocks*2 && number > 0 {
			go getBlockValues(number, sampleNumber, uint64(ignorePrice), result, quit, apiServant)
			sent++
			exp++
			number--
		}
		results = append(results, res.values...)
	}
	price := big.NewInt(int64(defaultGasPrice))
	if len(results) > 0 {
		price = results[(len(results)-1)*percentile/100]
	}
	if price.Cmp(maxPrice) > 0 {
		price.Set(maxPrice)
	}

	return price, nil
}
func getBlockValues(blockNum uint64, limit int, ignoreUnder uint64, result chan results, quit chan struct{}, apiServant servant.APIServant) {
	block, err := apiServant.GetBlockByHeight(blockNum)

	if block == nil {
		select {
		case result <- results{nil, err}:
		case <-quit:
		}
		return
	}
	txs := make([][]byte, block.GetHead().GetTxCount())
	copy(txs, block.GetData().GetTxs())
	transactions := make([]*txbasic.Transaction, 0)
	for i := 0; i < int(block.GetHead().GetTxCount()); i++ {
		tx, _ := apiServant.GetTransactionByHash(string(txs[i]))
		transactions = append(transactions, tx)
	}
	var prices []*big.Int
	for _, tx := range transactions {
		var transaction txuni.TransactionUniversal
		_ = json.Unmarshal(tx.GetData().Specification, &transaction)
		if transaction.GetHead().GetGasPrice() < ignoreUnder {
			continue
		}
		prices = append(prices, big.NewInt(int64(transaction.GetHead().GetGasPrice())))
		if len(prices) >= limit {
			break
		}
	}
	sort.Slice(prices, func(i, j int) bool {
		if prices[i].Cmp(prices[j]) > 0 {
			return true
		} else {
			return false
		}
	})
	select {
	case result <- results{prices, nil}:
	case <-quit:
	}
}

type results struct {
	values []*big.Int
	err    error
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
			var transactionUniversal txuni.TransactionUniversal
			_ = json.Unmarshal(transaction.GetData().GetSpecification(), &transactionUniversal)
			transactionReceipt, _ := servant.GetTransactionResultByHash(string(txHash))
			var transactionResultUniversal txuni.TransactionResultUniversal
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
	rewardStr := make([][]*hexutil.Big, len(Reward))
	baseFeeStr := make([]*hexutil.Big, len(BaseFee))

	for i, v := range Reward {
		for iv, vv := range v {
			rewardStr[i] = make([]*hexutil.Big, len(percentiles))
			rewardStr[i][iv] = (*hexutil.Big)(vv)
		}
	}
	for i, _ := range BaseFee {
		baseFeeStr[i] = (*hexutil.Big)(big.NewInt(0))
	}
	return &FeeHistoryResponseType{
		OldestBlock:  (*hexutil.Big)(new(big.Int).SetUint64(oldestBlock)),
		Reward:       rewardStr,
		BaseFee:      baseFeeStr,
		GasUsedRatio: GasUsedRatio,
	}
}
