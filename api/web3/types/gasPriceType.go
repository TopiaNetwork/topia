package types

import (
	"encoding/binary"
	"encoding/json"
	"github.com/TopiaNetwork/topia/api/servant"
	txbasic "github.com/TopiaNetwork/topia/transaction/basic"
	"github.com/TopiaNetwork/topia/transaction/universal"
	"math"
	"math/big"
	"math/rand"
	"sort"
	"time"
)

type GasPriceRequestType struct {
	Address string
	Height  string
}

type GasPriceResponseType struct {
	Balance string `json:"balance"`
}

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

type results struct {
	values []*big.Int
	err    error
}

func EstimateTipFee() (*big.Int, error) {
	servant := servant.NewAPIServant()
	block, _ := servant.GetLatestBlock()
	height := block.GetHead().GetHeight()

	var (
		sent, exp int
		number    = height
		result    = make(chan results, checkBlocks)
		quit      = make(chan struct{})
		results   []*big.Int
	)
	for sent < checkBlocks && number > 0 {
		go getBlockValues(number, sampleNumber, uint64(ignorePrice), result, quit)
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
			go getBlockValues(number, sampleNumber, uint64(ignorePrice), result, quit)
			sent++
			exp++
			number--
		}
		results = append(results, res.values...)
	}
	price := big.NewInt(int64(defaultGasPrice)) //如果采样值为空，则使用上一次的计算结果
	if len(results) > 0 {
		price = results[(len(results)-1)*percentile/100]
	}
	//估算tip时有一个最大值，500gwei
	if price.Cmp(maxPrice) > 0 {
		price.Set(maxPrice)
	}

	return price, nil
}

//没有做矿工交易筛选
func getBlockValues(blockNum uint64, limit int, ignoreUnder uint64, result chan results, quit chan struct{}) {
	servant := servant.NewAPIServant()
	block, err := servant.GetBlockByHeight(blockNum)

	if block == nil {
		select {
		case result <- results{nil, err}:
		case <-quit:
		}
		return
	}
	// Sort the transaction by effective tip in ascending sort.
	txs := make([][]byte, block.GetHead().GetTxCount())
	//获取当前block中的所有交易hash
	copy(txs, block.GetData().GetTxs())
	//获取所有的交易实例
	transactions := make([]*txbasic.Transaction, block.GetHead().GetTxCount())
	for i := 0; i < int(block.GetHead().GetTxCount()); i++ {
		tx, _ := servant.GetTransactionByHash(string(txs[i]))
		transactions = append(transactions, tx)
	}

	//提取所有的交易的gasPrice
	var prices []*big.Int
	for _, tx := range transactions {
		var transaction universal.TransactionUniversal
		_ = json.Unmarshal(tx.GetData().Specification, transaction)
		if transaction.GetHead().GetGasPrice() < ignoreUnder {
			continue
		}
		prices = append(prices, big.NewInt(int64(transaction.GetHead().GetGasPrice())))
		if len(prices) >= limit {
			break
		}
	}
	//对prices进行排序
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

//估算baseFee
func estimateBaseFee() uint64 {
	randRange := []float64{0.5, 0.8, 1.0, 1.5}
	rand.Seed(time.Now().Unix())
	index := rand.Intn(4)
	randomBaseFee := randRange[index] * float64(initBaseFee)

	//获取当前最新区块高度
	servant := servant.NewAPIServant()
	//调用方法
	block, _ := servant.GetLatestBlock()
	height := block.GetHead().GetHeight()
	//计算baseFee初始块高
	initHeight := height - uint64(window)
	for i := initHeight; i <= height; i++ {
		//获取指定块高的block，计算gasUsed/gasLimit
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
