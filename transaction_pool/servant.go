package transactionpool

import (
	"math/big"

	"github.com/TopiaNetwork/topia/account"
	"github.com/TopiaNetwork/topia/common/types"
	"github.com/TopiaNetwork/topia/network/p2p"
	"github.com/TopiaNetwork/topia/transaction"
)

type TransactionPoolServant interface {
	CurrentBlock() *types.Block
	GetBlock(hash types.BlockHash, num uint64) *types.Block
	StateAt(root types.BlockHash) (*StatePoolDB, error)
	SubChainHeadEvent(ch chan<- transaction.ChainHeadEvent) p2p.P2PPubSubService
	EstimateTxCost(tx *transaction.Transaction) *big.Int
	EstimateTxGas(tx *transaction.Transaction) uint64
	GetMaxGasLimit() uint64
}

type StatePoolDB struct {
}

func (st *StatePoolDB) GetAccount(addr account.Address) (*account.Account, error) {
	acc := &account.Account{
		Addr:    "",
		Name:    "",
		Nonce:   0,
		Balance: nil,
	}
	return acc, nil
}
func (st *StatePoolDB) GetBalance(addr account.Address) *big.Int {
	balance := big.NewInt(100000000)
	return balance
}

func (st *StatePoolDB) GetNonce(addr account.Address) uint64 {
	nonce := uint64(123456)
	return nonce
}
