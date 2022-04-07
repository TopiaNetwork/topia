package transactionpool

import (
	"encoding/json"
	tpcrtypes "github.com/TopiaNetwork/topia/crypt/types"
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

	PendingAccountSlots uint64 // Number of executable transaction slots guaranteed per account
	PendingGlobalSlots  uint64 // Maximum number of executable transaction slots for all accounts
	QueueMaxTxsAccount  uint64 // Maximum number of non-executable transaction slots permitted per account
	QueueMaxTxsGlobal   uint64 // Maximum number of non-executable transaction slots for all accounts

	LifetimeForTx         time.Duration
	DurationForTxRePublic time.Duration
}

var DefaultTransactionPoolConfig = TransactionPoolConfig{
	PathLocal: map[basic.TransactionCategory]string{
		basic.TransactionCategory_Topia_Universal: "Topia_Universal_localTransactions.json",
		basic.TransactionCategory_Eth:             "Eth_localTransactions.json"},
	PathRemote: map[basic.TransactionCategory]string{
		basic.TransactionCategory_Topia_Universal: "Topia_Universal_remoteTransactions.json",
		basic.TransactionCategory_Eth:             "Eth_remoteTransactions.json"},
	PathConfig:  "txPoolConfigs.json",
	ReStoredDur: 30 * time.Minute,

	GasPriceLimit:       1000,     // 1000
	PendingAccountSlots: 16,       //16
	PendingGlobalSlots:  8192,     //8192
	QueueMaxTxsAccount:  64,       //64
	QueueMaxTxsGlobal:   8192 * 2, //PendingGlobalSlots*2

	LifetimeForTx:         30 * time.Minute,
	DurationForTxRePublic: 30 * time.Second,
}

func (config *TransactionPoolConfig) check() TransactionPoolConfig {
	conf := *config
	if conf.GasPriceLimit < 1 {
		//tplog.Logger.Infof("Invalid GasPriceLimit,updated to default value:", "from", conf.GasPriceLimit, "to", DefaultTransactionPoolConfig.GasPriceLimit)
		conf.GasPriceLimit = DefaultTransactionPoolConfig.GasPriceLimit
	}
	if conf.PendingAccountSlots < 1 {
		//tplog.Logger.Infof("Invalid PendingAccountSlots,updated to default value:", "from", conf.PendingAccountSlots, "to", DefaultTransactionPoolConfig.PendingAccountSlots)
		conf.PendingAccountSlots = DefaultTransactionPoolConfig.PendingAccountSlots
	}
	if conf.PendingGlobalSlots < 1 {
		//tplog.Logger.Infof("Invalid PendingGlobalSlots,updated to default value:", "from", conf.PendingGlobalSlots, "to", DefaultTransactionPoolConfig.PendingGlobalSlots)
		conf.PendingGlobalSlots = DefaultTransactionPoolConfig.PendingGlobalSlots
	}
	if conf.QueueMaxTxsAccount < 1 {
		//tplog.Logger.Infof("Invalid QueueMaxTxsAccount,updated to default value:", "from", conf.QueueMaxTxsAccount, "to", DefaultTransactionPoolConfig.QueueMaxTxsAccount)
		conf.QueueMaxTxsAccount = DefaultTransactionPoolConfig.QueueMaxTxsAccount
	}
	if conf.QueueMaxTxsGlobal < 1 {
		//tplog.Logger.Infof("Invalid QueueMaxTxsGlobal,updated to default value:", "from", conf.QueueMaxTxsGlobal, "to", DefaultTransactionPoolConfig.QueueMaxTxsGlobal)
		conf.QueueMaxTxsGlobal = DefaultTransactionPoolConfig.QueueMaxTxsGlobal
	}
	if conf.LifetimeForTx < 1 {
		conf.LifetimeForTx = DefaultTransactionPoolConfig.LifetimeForTx
	}
	if conf.DurationForTxRePublic < 1 {
		conf.DurationForTxRePublic = DefaultTransactionPoolConfig.DurationForTxRePublic
	}
	if conf.PathConfig == "" {
		conf.PathConfig = DefaultTransactionPoolConfig.PathConfig
	}
	if conf.PathRemote == nil {
		conf.PathRemote = DefaultTransactionPoolConfig.PathRemote
	}
	if conf.PathLocal == nil {
		conf.PathLocal = DefaultTransactionPoolConfig.PathLocal
	}

	return conf
}

func (pool *transactionPool) LoadConfig() (conf *TransactionPoolConfig, error error) {
	data, err := ioutil.ReadFile(pool.config.PathConfig)
	if err != nil {
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
