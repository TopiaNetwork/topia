package transactionpool

import (
	"errors"
	"time"

	tpchaintypes "github.com/TopiaNetwork/topia/chain/types"
	txpooli "github.com/TopiaNetwork/topia/transaction_pool/interface"
)

type TxExpiredPolicy byte

const (
	TxExpiredTime TxExpiredPolicy = iota
	TxExpiredHeight
	TxExpiredTimeAndHeight
	TxExpiredTimeOrHeight
)

type PickTxType uint32

const (
	PickTxPending PickTxType = iota
	PickTxPriceAndNonce
	PickTxPriceAndTime
)

const (
	ChanBlockAddedSize   = 64
	ChanBlocksRevertSize = 64
	ChanDelTxsStorage    = 20 * 1024
	ChanSaveTxsStorage   = 20 * 1024
	TxCacheSize          = 36000000
	MaxUint64            = 1<<64 - 1
)

var (
	RemoveTxInterval         = 15 * time.Second // Time interval to check for remove transactions
	SaveTxStorageInterval    = 50 * time.Millisecond
	delTxFromStorageInterval = 100 * time.Millisecond
	RepublishTxInterval      = 3 * time.Second //30000 * time.Millisecond  //time interval to check transaction lifetime for report

	ObsID string

	ErrAlreadyKnown     = errors.New("transaction is already know")
	ErrTxNotExist       = errors.New("transaction not found")
	ErrAddrNotExist     = errors.New("address not found")
	ErrTxsNotContinuous = errors.New("transactions in pending is not continuous")
	ErrNonceNotMin      = errors.New("nonce is not min when remove tx")
	ErrTxIDDiff         = errors.New("TxID is different for the same nonce ")
	ErrTxIsPackaged     = errors.New("transaction is packaged can't be replaced")
	ErrTxIsNil          = errors.New("transaction is nil")
	ErrUnRooted         = errors.New("UnRooted new chain")
	ErrTxStoragePath    = errors.New("error tx storage path")
	ErrTxPoolFull       = errors.New("tx pool is full")
)

type BlockAddedEvent struct{ Block *tpchaintypes.Block }
type BlocksRevertEvent struct {
	Blocks []*tpchaintypes.Block
}

type TxRepublishPolicy byte

const (
	TxRepublishTime TxRepublishPolicy = iota
	TxRepublishHeight
	TxRepublishTimeAndHeight
	TxRepublishTimeOrHeight
)

func (pool *transactionPool) SetTxPoolConfig(conf txpooli.TransactionPoolConfig) {
	conf = (conf).Check()
	pool.config = conf
	return
}
