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
)

type TxExpiredPolicy byte

const (
	TxExpiredTime TxExpiredPolicy = iota
	TxExpiredHeight
	TxExpiredTimeAndHeight
	TxExpiredTimeOrHeight
)

type TransactionPoolConfig struct {
	NoRemoteFile    bool
	NoConfigFile    bool
	PathRemote      map[txbasic.TransactionCategory]string
	PathConfig      string
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
	PathRemote: map[txbasic.TransactionCategory]string{
		txbasic.TransactionCategory_Topia_Universal: "savedtxs/Topia_Universal_remoteTransactions.json",
		txbasic.TransactionCategory_Eth:             "savedtxs/Eth_remoteTransactions.json"},
	PathConfig:      "configuration/txPoolConfigs.json",
	ReStoredDur:     30 * time.Minute,
	TxExpiredPolicy: TxExpiredTimeAndHeight,
	GasPriceLimit:   1000, // 1000
	TxCacheSize:     36000000,

	TxMaxSize:     4 * 1024,
	TxPoolMaxSize: 16 * 4 * 1024,

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
	if conf.MaxSizeOfEachPendingAccount < 32*1024 {
		conf.MaxSizeOfEachQueueAccount = DefaultTransactionPoolConfig.MaxSizeOfEachPendingAccount
	}
	if conf.MaxSizeOfPending < 16*32*1024 {
		conf.MaxSizeOfPending = DefaultTransactionPoolConfig.MaxSizeOfPending
	}
	if conf.MaxSizeOfEachQueueAccount < 64*1024 {
		conf.MaxSizeOfEachPendingAccount = DefaultTransactionPoolConfig.MaxSizeOfEachQueueAccount
	}
	if conf.MaxSizeOfQueue < 32*64*1024 {
		conf.MaxSizeOfQueue = DefaultTransactionPoolConfig.MaxSizeOfQueue
	}
	if conf.LifetimeForTx < 1 {
		conf.LifetimeForTx = DefaultTransactionPoolConfig.LifetimeForTx
	}
	if conf.TxTTLTimeOfRepublic < 1 {
		conf.TxTTLTimeOfRepublic = DefaultTransactionPoolConfig.TxTTLTimeOfRepublic
	}
	if conf.PathConfig == "" {
		conf.PathConfig = DefaultTransactionPoolConfig.PathConfig
	}
	if conf.PathRemote == nil {
		conf.PathRemote = DefaultTransactionPoolConfig.PathRemote
	}

	return conf
}

type TransactionPool interface {
	AddTx(tx *txbasic.Transaction, local bool) error

	RemoveTxByKey(key txbasic.TxID) error

	RemoveTxHashs(hashs []txbasic.TxID) []error

	Reset(oldHead, newHead *tpchaintypes.BlockHead) error

	UpdateTx(tx *txbasic.Transaction, txKey txbasic.TxID) error

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
