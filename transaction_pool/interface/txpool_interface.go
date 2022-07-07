package txpoolinterface

import (
	"time"

	"github.com/AsynkronIT/protoactor-go/actor"

	tpcrtypes "github.com/TopiaNetwork/topia/crypt/types"
	tpnet "github.com/TopiaNetwork/topia/network"
	txbasic "github.com/TopiaNetwork/topia/transaction/basic"
)

const MOD_NAME = "TransactionPool"

type TransactionState string

const (
	StateTxAdded       TransactionState = "transaction Added"
	StateTxRepublished                  = "transaction republished"
	StateTxNil                          = "no transaction state for this tx "
)

type TransactionPoolConfig struct {
	PathTxsStorage string
	PathConf       string
	ReStoredDur    time.Duration
	IsLoadTxs      bool
	IsLoadCfg      bool
	GasPriceLimit  uint64

	MaxSizeOfEachTx     int64
	TxPoolMaxSize       int64
	TxPoolMaxCnt        int64
	MaxCntOfEachAccount int64 // Maximum size of transaction per account

	LifetimeForTx          time.Duration
	LifeHeight             uint64
	TxTTLTimeOfRepublish   time.Duration
	TxTTLHeightOfRepublish uint64
}

var DefaultTransactionPoolConfig = TransactionPoolConfig{
	PathTxsStorage: "StorageInfo/StorageTxs",
	PathConf:       "StorageInfo/StorageConfig.json",
	ReStoredDur:    120 * time.Second, //30 * time.Minute,

	GasPriceLimit: 1000, // 1000

	MaxSizeOfEachTx:     2 * 1024,
	TxPoolMaxSize:       20 * 1024 * 1024,
	TxPoolMaxCnt:        40 * 1024,
	MaxCntOfEachAccount: 32,

	LifetimeForTx:          15 * time.Minute, //15 * time.Minute,
	LifeHeight:             uint64(30 * 60),
	TxTTLTimeOfRepublish:   30 * time.Second, //30011 * time.Millisecond, //Prime Numbers 30second
	TxTTLHeightOfRepublish: 30,
}

func (config *TransactionPoolConfig) Check() TransactionPoolConfig {
	conf := *config
	if conf.GasPriceLimit < DefaultTransactionPoolConfig.GasPriceLimit {
		conf.GasPriceLimit = DefaultTransactionPoolConfig.GasPriceLimit
	}
	if conf.MaxCntOfEachAccount < DefaultTransactionPoolConfig.MaxCntOfEachAccount {
		conf.MaxCntOfEachAccount = DefaultTransactionPoolConfig.MaxCntOfEachAccount
	}

	if conf.TxPoolMaxSize < DefaultTransactionPoolConfig.TxPoolMaxSize {
		conf.TxPoolMaxSize = DefaultTransactionPoolConfig.TxPoolMaxSize
	}
	if conf.TxPoolMaxCnt < DefaultTransactionPoolConfig.TxPoolMaxCnt {
		conf.TxPoolMaxCnt = DefaultTransactionPoolConfig.TxPoolMaxCnt
	}
	if conf.LifetimeForTx < DefaultTransactionPoolConfig.LifetimeForTx {
		conf.LifetimeForTx = DefaultTransactionPoolConfig.LifetimeForTx
	}
	if conf.TxTTLTimeOfRepublish < DefaultTransactionPoolConfig.TxTTLTimeOfRepublish {
		conf.TxTTLTimeOfRepublish = DefaultTransactionPoolConfig.TxTTLTimeOfRepublish
	}
	if conf.TxTTLHeightOfRepublish < DefaultTransactionPoolConfig.TxTTLHeightOfRepublish {
		conf.TxTTLHeightOfRepublish = DefaultTransactionPoolConfig.TxTTLHeightOfRepublish
	}
	if conf.PathTxsStorage == "" {
		conf.PathTxsStorage = DefaultTransactionPoolConfig.PathTxsStorage
	}

	return conf
}

type TransactionPool interface {
	AddTx(tx *txbasic.Transaction, isLocal bool) error

	RemoveTxByKey(key txbasic.TxID, force bool) error

	RemoveTxHashes(hashes []txbasic.TxID, force bool) error

	UpdateTx(tx *txbasic.Transaction, txKey txbasic.TxID) error

	PendingOfAddress(addr tpcrtypes.Address) ([]*txbasic.Transaction, error)

	PickTxs() []*txbasic.Transaction

	Count() int64

	Size() int64

	TruncateTxPool()

	Start(sysActor *actor.ActorSystem, network tpnet.Network) error

	SysShutDown()

	SetTxPoolConfig(conf TransactionPoolConfig)

	PeekTxState(hash txbasic.TxID) TransactionState
}
