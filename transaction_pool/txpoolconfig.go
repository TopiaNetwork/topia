package transactionpool

import (
	"errors"
	tpconfig "github.com/TopiaNetwork/topia/configuration"
	"time"

	tpchaintypes "github.com/TopiaNetwork/topia/chain/types"
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
	ChanAddTxsSize        = 1024 * 1024
	ChanBlockAddedSize    = 64
	ChanDelTxsStorage     = 20 * 1024
	ChanSaveTxsStorage    = 20 * 1024
	TxCacheSize           = 1024 * 1024
	AccountNonceCacheSize = 40 * 1024
	MaxUint64             = 1<<64 - 1
)

var (
	RemoveTxInterval         = 30011 * time.Millisecond // Time interval to check for remove transactions
	SaveTxStorageInterval    = 101 * time.Millisecond
	delTxFromStorageInterval = 151 * time.Millisecond
	RepublishTxInterval      = 1499 * time.Millisecond //30000 * time.Millisecond  //time interval to check transaction lifetime for report

	ObsID string

	ErrAlreadyKnown  = errors.New("transaction is already know")
	ErrTxNotExist    = errors.New("transaction not found")
	ErrAddrNotExist  = errors.New("address not found")
	ErrNoTxAdded     = errors.New("no transactions add to txPool")
	ErrNonceNotMin   = errors.New("nonce is not min when remove tx")
	ErrTxIDDiff      = errors.New("TxID is different for the same nonce ")
	ErrTxIsPackaged  = errors.New("transaction is packaged can't be replaced")
	ErrTxIsNil       = errors.New("transaction is nil")
	ErrUnRooted      = errors.New("UnRooted new chain")
	ErrTxStoragePath = errors.New("error tx storage path")
	ErrTxPoolFull    = errors.New("tx pool is full")
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

func (pool *transactionPool) SetTxPoolConfig(conf *tpconfig.TransactionPoolConfig) {
	confNew := conf.Check()
	pool.config = confNew
	return
}
