package transactionpool

import (
	"encoding/json"
	tpcrtypes "github.com/TopiaNetwork/topia/crypt/types"
	tplog "github.com/TopiaNetwork/topia/log"
	"github.com/TopiaNetwork/topia/transaction/basic"
	"io/ioutil"
	"time"
)

type TransactionPoolConfig struct {
	Locals       []tpcrtypes.Address
	NoLocalFile  bool
	NoRemoteFile bool
	NoConfigFile bool
	PathLocal    map[basic.TransactionCategory]string
	PathRemote   map[basic.TransactionCategory]string
	PathConfig   string
	ReStoredDur  time.Duration

	GasPriceLimit uint64

	PendingAccountSegments uint64 // Number of executable transaction slots guaranteed per account
	PendingGlobalSegments  uint64 // Maximum number of executable transaction slots for all accounts
	QueueMaxTxsAccount     uint64 // Maximum number of non-executable transaction slots permitted per account
	QueueMaxTxsGlobal      uint64 // Maximum number of non-executable transaction slots for all accounts

	LifetimeForTx         time.Duration
	DurationForTxRePublic time.Duration
	EvictionInterval      time.Duration //= 29989 * time.Millisecond // Time interval to check for evictable transactions
	StatsReportInterval   time.Duration //= 499 * time.Millisecond // Time interval to report transaction pool stats
	RepublicInterval      time.Duration //= 30011 * time.Millisecond       //time interval to check transaction lifetime for report
}

var DefaultTransactionPoolConfig = TransactionPoolConfig{
	PathLocal: map[basic.TransactionCategory]string{
		basic.TransactionCategory_Topia_Universal: "savedtxs/Topia_Universal_localTransactions.json",
		basic.TransactionCategory_Eth:             "savedtxs/Eth_localTransactions.json"},
	PathRemote: map[basic.TransactionCategory]string{
		basic.TransactionCategory_Topia_Universal: "savedtxs/Topia_Universal_remoteTransactions.json",
		basic.TransactionCategory_Eth:             "savedtxs/Eth_remoteTransactions.json"},
	PathConfig:  "configuration/txPoolConfigs.json",
	ReStoredDur: 30 * time.Minute,

	GasPriceLimit:          1000,     // 1000
	PendingAccountSegments: 64,       //64
	PendingGlobalSegments:  8192,     //8192
	QueueMaxTxsAccount:     64,       //64
	QueueMaxTxsGlobal:      8192 * 2, //PendingGlobalSlots*2

	LifetimeForTx:         30 * time.Minute,
	DurationForTxRePublic: 30011 * time.Millisecond, //Prime Numbers 30second
	EvictionInterval:      30013 * time.Millisecond, // Time interval to check for evictable transactions
	StatsReportInterval:   499 * time.Millisecond,   // Time interval to report transaction pool stats
	RepublicInterval:      30029 * time.Millisecond, //time interval to check transaction lifetime for report

}

func (config *TransactionPoolConfig) check() TransactionPoolConfig {
	conf := *config
	if conf.GasPriceLimit < 1 {
		tplog.Logger.Warnf("Invalid GasPriceLimit,updated to default value:", "from", conf.GasPriceLimit, "to", DefaultTransactionPoolConfig.GasPriceLimit)
		conf.GasPriceLimit = DefaultTransactionPoolConfig.GasPriceLimit
	}
	if conf.PendingAccountSegments < 2 {
		tplog.Logger.Warnf("Invalid PendingAccountSlots,updated to default value:", "from", conf.PendingAccountSegments, "to", DefaultTransactionPoolConfig.PendingAccountSegments)
		conf.PendingAccountSegments = DefaultTransactionPoolConfig.PendingAccountSegments
	}
	if conf.PendingGlobalSegments < 1 {
		tplog.Logger.Warnf("Invalid PendingGlobalSlots,updated to default value:", "from", conf.PendingGlobalSegments, "to", DefaultTransactionPoolConfig.PendingGlobalSegments)
		conf.PendingGlobalSegments = DefaultTransactionPoolConfig.PendingGlobalSegments
	}
	if conf.QueueMaxTxsAccount < 1 {
		tplog.Logger.Warnf("Invalid QueueMaxTxsAccount,updated to default value:", "from", conf.QueueMaxTxsAccount, "to", DefaultTransactionPoolConfig.QueueMaxTxsAccount)
		conf.QueueMaxTxsAccount = DefaultTransactionPoolConfig.QueueMaxTxsAccount
	}
	if conf.QueueMaxTxsGlobal < 1 {
		tplog.Logger.Warnf("Invalid QueueMaxTxsGlobal,updated to default value:", "from", conf.QueueMaxTxsGlobal, "to", DefaultTransactionPoolConfig.QueueMaxTxsGlobal)
		conf.QueueMaxTxsGlobal = DefaultTransactionPoolConfig.QueueMaxTxsGlobal
	}
	if conf.LifetimeForTx < 1 {
		tplog.Logger.Warnf("Invalid LifetimeForTx,updated to default value:", "from", conf.LifetimeForTx, "to", DefaultTransactionPoolConfig.LifetimeForTx)
		conf.LifetimeForTx = DefaultTransactionPoolConfig.LifetimeForTx
	}
	if conf.DurationForTxRePublic < 1 {
		tplog.Logger.Warnf("Invalid DurationForTxRePublic,updated to default value:", "from", conf.DurationForTxRePublic, "to", DefaultTransactionPoolConfig.DurationForTxRePublic)
		conf.DurationForTxRePublic = DefaultTransactionPoolConfig.DurationForTxRePublic
	}
	if conf.PathConfig == "" {
		tplog.Logger.Warnf("Invalid PathConfig,updated to default value:", "from", conf.PathConfig, "to", DefaultTransactionPoolConfig.PathConfig)
		conf.PathConfig = DefaultTransactionPoolConfig.PathConfig
	}
	if conf.PathRemote == nil {
		tplog.Logger.Warnf("Invalid PathRemote,updated to default value:", "from", conf.PathRemote, "to", DefaultTransactionPoolConfig.PathRemote)
		conf.PathRemote = DefaultTransactionPoolConfig.PathRemote
	}
	if conf.PathLocal == nil {
		tplog.Logger.Warnf("Invalid PathLocal,updated to default value:", "from", conf.PathLocal, "to", DefaultTransactionPoolConfig.PathLocal)
		conf.PathLocal = DefaultTransactionPoolConfig.PathLocal
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

func (pool *transactionPool) UpdateTxPoolConfig(conf TransactionPoolConfig) {
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
