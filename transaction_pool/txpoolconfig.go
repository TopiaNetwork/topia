package transactionpool

import (
	"encoding/json"
	"io/ioutil"
	"time"

	txbasic "github.com/TopiaNetwork/topia/transaction/basic"
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
	TxpoolMaxSize   uint64

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

	TxCacheSize: 36000000,

	TxMaxSize:     4 * 32 * 1024,
	TxpoolMaxSize: 8192 * 2 * 4 * 32 * 1024,

	MaxSizeOfEachPendingAccount: 64 * 32 * 1024,
	MaxSizeOfPending:            8192 * 32 * 1024,
	MaxSizeOfEachQueueAccount:   64 * 32 * 1024,
	MaxSizeOfQueue:              8192 * 2 * 32 * 1024,

	LifetimeForTx:         30 * time.Minute,
	LifeHeight:            uint64(30 * 60),
	TxTTLTimeOfRepublic:   30011 * time.Millisecond, //Prime Numbers 30second
	TxTTLHeightOfRepublic: 30,
	EvictionInterval:      30013 * time.Millisecond, // Time interval to check for evictable transactions
	RepublicInterval:      30029 * time.Millisecond, //time interval to check transaction lifetime for report

}

func (config *TransactionPoolConfig) check() TransactionPoolConfig {
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

func (pool *transactionPool) LoadConfig() (conf *TransactionPoolConfig, error error) {
	data, err := ioutil.ReadFile(pool.config.PathConfig)
	if err != nil {
		pool.log.Warnf("Failed to load transaction config from stored file", "err", err)
		return nil, err
	}
	config := &conf
	err = json.Unmarshal(data, &config)
	if err != nil {
		return nil, err
	}
	return *config, nil
}

func (pool *transactionPool) SetTxPoolConfig(conf TransactionPoolConfig) {
	conf = (conf).check()
	pool.config = conf
	return
}

func (pool *transactionPool) SaveConfig() error {
	//pool.log.Info("saving tx pool config")
	conf, err := json.Marshal(pool.config)
	if err != nil {
		return err
	}

	err = ioutil.WriteFile(pool.config.PathConfig, conf, 0664)
	if err != nil {
		return err
	}
	return nil
}

func (pool *transactionPool) loadConfig(nofile bool, path string) {
	if !nofile && path != "" {
		if con, err := pool.LoadConfig(); err != nil {
			pool.log.Warnf("Failed to load txPool configs", "err", err)
		} else {
			pool.config = *con
		}
	}
}
