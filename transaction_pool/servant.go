package transactionpool

import (
	tpcrtypes "github.com/TopiaNetwork/topia/crypt/types"
	"math/big"

	"github.com/TopiaNetwork/topia/account"
	"github.com/TopiaNetwork/topia/chain/types"
	"github.com/TopiaNetwork/topia/network/p2p"
	"github.com/TopiaNetwork/topia/transaction/basic"
)

type ChainHeadEvent struct{ Block *types.Block }
type TransactionPoolServant interface {
	CurrentBlock() *types.Block
	GetBlock(hash types.BlockHash, num uint64) *types.Block
	StateAt(root types.BlockHash) (*StatePoolDB, error)
	SubChainHeadEvent(ch chan<- ChainHeadEvent) p2p.P2PPubSubService
	EstimateTxCost(tx *basic.Transaction) *big.Int
	EstimateTxGas(tx *basic.Transaction) uint64
	GetMaxGasLimit() uint64
}

type StatePoolDB struct {
}

func (st *StatePoolDB) GetAccount(addr tpcrtypes.Address) (*account.Account, error) {
	acc := &account.Account{
		Addr:    "",
		Name:    "",
		Nonce:   0,
		Balance: nil,
	}
	return acc, nil
}
func (st *StatePoolDB) GetBalance(addr tpcrtypes.Address) *big.Int {
	balance := big.NewInt(100000000)
	return balance
}

func (st *StatePoolDB) GetNonce(addr tpcrtypes.Address) uint64 {
	nonce := uint64(123456)
	return nonce
}
