package configuration

import "time"

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
	PendingMaxSize      int64
	PendingMaxCnt       int64
	MaxCntOfEachAccount int64 // Maximum size of transaction per account

	BlockMaxBytes int64
	BlockMaxGas   int64

	TimeOfTxLifecycle      time.Duration
	HeightOfTxLifecycle    uint64
	TimeOfTxRepublishTTL   time.Duration
	HeightOfTxRepublishTTL uint64
}

func DefaultTransactionPoolConfig() *TransactionPoolConfig {
	return &TransactionPoolConfig{
		PathTxsStorage: "StorageInfo/StorageTxs",
		PathConf:       "StorageInfo/StorageConfig.json",
		ReStoredDur:    2000003 * time.Microsecond, //2 * time.Second,

		GasPriceLimit: 1000, // 1000

		MaxSizeOfEachTx: 2 * 1024,
		TxPoolMaxSize:   20 * 1024 * 1024,
		TxPoolMaxCnt:    40 * 1024,
		PendingMaxSize:  10 * 1024 * 1024,
		PendingMaxCnt:   20 * 1024,

		MaxCntOfEachAccount: 32,

		BlockMaxBytes: 22020096, //21MB
		BlockMaxGas:   -1,

		TimeOfTxLifecycle:      15 * time.Second, // 15 * time.Second
		HeightOfTxLifecycle:    uint64(30 * 60),
		TimeOfTxRepublishTTL:   5 * time.Second, //5 * time.Second
		HeightOfTxRepublishTTL: 5,
	}
}

func (config *TransactionPoolConfig) Check() *TransactionPoolConfig {
	conf := *config
	if conf.GasPriceLimit < DefaultTransactionPoolConfig().GasPriceLimit {
		conf.GasPriceLimit = DefaultTransactionPoolConfig().GasPriceLimit
	}
	if conf.MaxCntOfEachAccount < DefaultTransactionPoolConfig().MaxCntOfEachAccount {
		conf.MaxCntOfEachAccount = DefaultTransactionPoolConfig().MaxCntOfEachAccount
	}

	if conf.TxPoolMaxSize < DefaultTransactionPoolConfig().TxPoolMaxSize {
		conf.TxPoolMaxSize = DefaultTransactionPoolConfig().TxPoolMaxSize
	}
	if conf.TxPoolMaxCnt < DefaultTransactionPoolConfig().TxPoolMaxCnt {
		conf.TxPoolMaxCnt = DefaultTransactionPoolConfig().TxPoolMaxCnt
	}
	if conf.PendingMaxSize < DefaultTransactionPoolConfig().PendingMaxSize {
		conf.PendingMaxSize = DefaultTransactionPoolConfig().PendingMaxSize
	}
	if conf.PendingMaxCnt < DefaultTransactionPoolConfig().PendingMaxCnt {
		conf.PendingMaxCnt = DefaultTransactionPoolConfig().PendingMaxCnt
	}

	if conf.TimeOfTxLifecycle < DefaultTransactionPoolConfig().TimeOfTxLifecycle {
		conf.TimeOfTxLifecycle = DefaultTransactionPoolConfig().TimeOfTxLifecycle
	}
	if conf.TimeOfTxRepublishTTL < DefaultTransactionPoolConfig().TimeOfTxRepublishTTL {
		conf.TimeOfTxRepublishTTL = DefaultTransactionPoolConfig().TimeOfTxRepublishTTL
	}
	if conf.HeightOfTxRepublishTTL < DefaultTransactionPoolConfig().HeightOfTxRepublishTTL {
		conf.HeightOfTxRepublishTTL = DefaultTransactionPoolConfig().HeightOfTxRepublishTTL
	}
	if conf.PathTxsStorage == "" {
		conf.PathTxsStorage = DefaultTransactionPoolConfig().PathTxsStorage
	}

	return &conf
}
