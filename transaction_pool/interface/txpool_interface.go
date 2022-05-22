package txpoolinterface

import (
	"github.com/AsynkronIT/protoactor-go/actor"
	tpchaintypes "github.com/TopiaNetwork/topia/chain/types"
	tpcrtypes "github.com/TopiaNetwork/topia/crypt/types"
	tpnet "github.com/TopiaNetwork/topia/network"
	txbasic "github.com/TopiaNetwork/topia/transaction/basic"
	"time"
)

const MOD_NAME = "TransactionPool"

type PickTxType uint32

const (
	PickTransactionsFromPending PickTxType = iota
	PickTransactionsSortedByGasPriceAndNonce
	PickTransactionsSortedByGasPriceAndTime
)

type TxExpiredPolicy byte

const (
	TxExpiredTime TxExpiredPolicy = iota
	TxExpiredHeight
	TxExpiredTimeAndHeight
	TxExpiredTimeOrHeight
)

type TransactionPoolConfig struct {
	PathMapRemoteTxsByCategory map[txbasic.TransactionCategory]string
	PathConfigFile             string
	PathTxsInfoFile            string

	ReStoredDur     time.Duration
	TxExpiredPolicy TxExpiredPolicy
	TxCacheSize     int
	GasPriceLimit   uint64
	TxMaxSize       uint64
	TxPoolMaxSize   uint64

	MaxSizeOfEachPendingAccount uint64 // Maximum size of executable transaction per account
	MaxSizeOfPending            uint64 // Maximum size of executable transaction
	MaxSizeOfEachQueueAccount   uint64 // Maximum number of non-executable transaction slots permitted per account
	MaxSizeOfQueue              uint64 // Maximum number of non-executable transaction slots for all accounts

	LifetimeForTx         time.Duration
	LifeHeight            uint64
	TxTTLTimeOfRepublic   time.Duration
	TxTTLHeightOfRepublic uint64
	EvictionInterval      time.Duration //= 29989 * time.Millisecond // Time interval to check for evictable transactions
	RepublicInterval      time.Duration //= 30011 * time.Millisecond       //time interval to check transaction lifetime for report
}

var DefaultTransactionPoolConfig = TransactionPoolConfig{
	PathMapRemoteTxsByCategory: map[txbasic.TransactionCategory]string{
		txbasic.TransactionCategory_Topia_Universal: "savedtxs/Topia_Universal_remoteTransactions.json"},
	PathConfigFile:  "configuration/txPoolConfigs.json",
	PathTxsInfoFile: "savedtxs/TxsInfo.json",
	ReStoredDur:     30 * time.Minute,
	TxExpiredPolicy: TxExpiredTimeAndHeight,
	GasPriceLimit:   1000, // 1000
	TxCacheSize:     36000000,

	TxMaxSize:     2 * 1024,
	TxPoolMaxSize: 64 * 2 * 1024,

	MaxSizeOfEachPendingAccount: 2 * 1024,
	MaxSizeOfPending:            16 * 1024,
	MaxSizeOfEachQueueAccount:   2 * 1024,
	MaxSizeOfQueue:              32 * 1024,

	LifetimeForTx:         30 * time.Minute,
	LifeHeight:            uint64(30 * 60),
	TxTTLTimeOfRepublic:   30011 * time.Millisecond, //Prime Numbers 30second
	TxTTLHeightOfRepublic: 30,
	EvictionInterval:      30013 * time.Millisecond, // Time interval to check for evictable transactions
	RepublicInterval:      30029 * time.Millisecond, //time interval to check transaction lifetime for report

}

func (config *TransactionPoolConfig) Check() TransactionPoolConfig {
	conf := *config
	if conf.GasPriceLimit < 1 {
		conf.GasPriceLimit = DefaultTransactionPoolConfig.GasPriceLimit
	}
	if conf.MaxSizeOfEachPendingAccount < 2*1024 {
		conf.MaxSizeOfEachQueueAccount = DefaultTransactionPoolConfig.MaxSizeOfEachPendingAccount
	}
	if conf.MaxSizeOfPending < 32*1024 {
		conf.MaxSizeOfPending = DefaultTransactionPoolConfig.MaxSizeOfPending
	}
	if conf.MaxSizeOfEachQueueAccount < 2*1024 {
		conf.MaxSizeOfEachPendingAccount = DefaultTransactionPoolConfig.MaxSizeOfEachQueueAccount
	}
	if conf.MaxSizeOfQueue < 64*1024 {
		conf.MaxSizeOfQueue = DefaultTransactionPoolConfig.MaxSizeOfQueue
	}
	if conf.TxPoolMaxSize < 128*1024 {
		conf.TxPoolMaxSize = DefaultTransactionPoolConfig.TxPoolMaxSize
	}
	if conf.LifetimeForTx < 1 {
		conf.LifetimeForTx = DefaultTransactionPoolConfig.LifetimeForTx
	}
	if conf.TxTTLTimeOfRepublic < 1 {
		conf.TxTTLTimeOfRepublic = DefaultTransactionPoolConfig.TxTTLTimeOfRepublic
	}
	if conf.PathConfigFile == "" {
		conf.PathConfigFile = DefaultTransactionPoolConfig.PathConfigFile
	}
	if conf.PathMapRemoteTxsByCategory == nil {
		conf.PathMapRemoteTxsByCategory = DefaultTransactionPoolConfig.PathMapRemoteTxsByCategory
	}
	if conf.PathTxsInfoFile == "" {
		conf.PathTxsInfoFile = DefaultTransactionPoolConfig.PathTxsInfoFile
	}

	return conf
}

type TransactionPool interface {
	AddTx(tx *txbasic.Transaction, local bool) error

	RemoveTxByKey(key txbasic.TxID) error

	RemoveTxHashs(hashs []txbasic.TxID) []error

	Reset(oldHead, newHead *tpchaintypes.BlockHead) error

	UpdateTx(tx *txbasic.Transaction, txKey txbasic.TxID, isLocal bool) error

	Pending() ([]*txbasic.Transaction, error)

	PendingOfAddress(addr tpcrtypes.Address) ([]*txbasic.Transaction, error)

	PickTxs(txType PickTxType) []*txbasic.Transaction

	Count() int64

	Size() int64

	TruncateTxPool()

	Start(sysActor *actor.ActorSystem, network tpnet.Network) error

	SysShutDown()

	SetTxPoolConfig(conf TransactionPoolConfig)

	PeekTxState(hash txbasic.TxID) interface{}
}
