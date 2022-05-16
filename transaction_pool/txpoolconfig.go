package transactionpool

import (
	"encoding/json"
	"io/ioutil"
	"time"

	tpcrtypes "github.com/TopiaNetwork/topia/crypt/types"
	txbasic "github.com/TopiaNetwork/topia/transaction/basic"
)

type TransactionPoolConfig struct {
	Locals               []tpcrtypes.Address
	NoLocalFile          bool
	NoRemoteFile         bool
	NoConfigFile         bool
	PathLocal            map[txbasic.TransactionCategory]string
	PathRemote           map[txbasic.TransactionCategory]string
	PathConfig           string
	ReStoredDur          time.Duration
	TxExpiredPolicy      TxExpiredPolicy
	TxStateCap           int
	GasPriceLimit        uint64
	TxSegmentSize        int
	TxMaxSegmentSize     int
	TxpoolMaxSegmentSize int

	PendingAccountSegments uint64 // Number of executable transaction slots guaranteed per account
	PendingGlobalSegments  uint64 // Maximum number of executable transaction slots for all accounts
	QueueMaxTxsAccount     uint64 // Maximum number of non-executable transaction slots permitted per account
	QueueMaxTxsGlobal      uint64 // Maximum number of non-executable transaction slots for all accounts

	LifetimeForTx           time.Duration
	LifeHeight              uint64
	DurationForTxRePublic   time.Duration
	DiffHeightForTxRePublic uint64
	EvictionInterval        time.Duration //= 29989 * time.Millisecond // Time interval to check for evictable transactions
	StatsReportInterval     time.Duration //= 499 * time.Millisecond // Time interval to report transaction pool stats
	RepublicInterval        time.Duration //= 30011 * time.Millisecond       //time interval to check transaction lifetime for report
}

var DefaultTransactionPoolConfig = TransactionPoolConfig{
	PathLocal: map[txbasic.TransactionCategory]string{
		txbasic.TransactionCategory_Topia_Universal: "savedtxs/Topia_Universal_localTransactions.json",
		txbasic.TransactionCategory_Eth:             "savedtxs/Eth_localTransactions.json"},
	PathRemote: map[txbasic.TransactionCategory]string{
		txbasic.TransactionCategory_Topia_Universal: "savedtxs/Topia_Universal_remoteTransactions.json",
		txbasic.TransactionCategory_Eth:             "savedtxs/Eth_remoteTransactions.json"},
	PathConfig:      "configuration/txPoolConfigs.json",
	ReStoredDur:     30 * time.Minute,
	TxExpiredPolicy: TxExpiredTimeAndHeight,
	GasPriceLimit:   1000, // 1000

	TxStateCap: 36000000,

	TxSegmentSize:        32 * 1024,
	TxMaxSegmentSize:     4 * 32 * 1024,
	TxpoolMaxSegmentSize: 8192 * 2 * 4 * 32 * 1024,

	PendingAccountSegments: 64,       //64
	PendingGlobalSegments:  8192,     //8192
	QueueMaxTxsAccount:     64,       //64
	QueueMaxTxsGlobal:      8192 * 2, //PendingGlobalSlots*2

	LifetimeForTx:           30 * time.Minute,
	LifeHeight:              uint64(30 * 60),
	DurationForTxRePublic:   30011 * time.Millisecond, //Prime Numbers 30second
	DiffHeightForTxRePublic: 30,
	EvictionInterval:        30013 * time.Millisecond, // Time interval to check for evictable transactions
	StatsReportInterval:     499 * time.Millisecond,   // Time interval to report transaction pool stats
	RepublicInterval:        30029 * time.Millisecond, //time interval to check transaction lifetime for report

}

func (config *TransactionPoolConfig) check() TransactionPoolConfig {
	conf := *config
	if conf.GasPriceLimit < 1 {
		conf.GasPriceLimit = DefaultTransactionPoolConfig.GasPriceLimit
	}
	if conf.PendingAccountSegments < 2 {
		conf.PendingAccountSegments = DefaultTransactionPoolConfig.PendingAccountSegments
	}
	if conf.PendingGlobalSegments < 1 {
		conf.PendingGlobalSegments = DefaultTransactionPoolConfig.PendingGlobalSegments
	}
	if conf.QueueMaxTxsAccount < 1 {
		conf.QueueMaxTxsAccount = DefaultTransactionPoolConfig.QueueMaxTxsAccount
	}
	if conf.QueueMaxTxsGlobal < 1 {
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
