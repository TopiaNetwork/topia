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
		transactions = append(transactions, tx) //TODO:这里数据没写进去呗
	}
	var prices []*big.Int
	for _, tx := range transactions {
		var transaction universal.TransactionUniversal
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
