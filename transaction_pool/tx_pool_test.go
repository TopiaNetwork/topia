package transactionpool

import (
	"github.com/TopiaNetwork/topia/account"
	"github.com/TopiaNetwork/topia/codec"
	tpcmm "github.com/TopiaNetwork/topia/common"
	"github.com/TopiaNetwork/topia/log"
	tplogcmm "github.com/TopiaNetwork/topia/log/common"
	"github.com/TopiaNetwork/topia/network"
	"github.com/TopiaNetwork/topia/network/p2p"
	"github.com/TopiaNetwork/topia/transaction"

	"sync"
	"testing"
	"time"
)

func settransaction() *transaction.Transaction {
	tx := &transaction.Transaction{
		FromAddr:   []byte{0x21, 0x23, 0x43, 0x53, 0x23, 0x34, 0x21, 0x12, 0x42, 0x12, 0x43},
		TargetAddr: []byte{0x23, 0x34, 0x34}, Version: 1, ChainID: []byte{0x01, 0x32, 0x46, 0x32, 0x32}, Nonce: 1, Value: []byte{0x12, 0x32}, GasPrice: 10000,
		GasLimit: 100, Data: []byte{0x32, 0x32, 0x32},
		Signature: []byte{0x32, 0x23, 0x42}, Options: 321}
	return tx
}
func settxpoolconfig() *TransactionPoolConfig {
	conf := &TransactionPoolConfig{
		chain: TransactionPoolServant{
			//demo
		},
		Locals: []account.Address{
			account.Address("2fadf9192731273"),
			account.Address("3fadfa123123123"),
			account.Address("31313131dsaf")},
		NoLocalFile:           false,
		NoRemoteFile:          false,
		NoConfigFile:          false,
		PathLocal:             "",
		PathRemote:            "",
		PathConfig:            "",
		ReStoredDur:           0,
		GasPriceLimit:         0,
		PendingAccountSlots:   0,
		PendingGlobalSlots:    0,
		QueueMaxTxsAccount:    0,
		QueueMaxTxsGlobal:     0,
		LifetimeForTx:         0,
		DurationForTxRePublic: 0,
	}
	return conf
}

func setupTxPool() *transactionPool {
	conf := settxpoolconfig
	log := log.Logger()
	newPool := NewTransactionPool(conf, 1)
	return newPool
}
func Test_transactionPool_AddTx(t *testing.T) {

	type fields struct {
		config              TransactionPoolConfig
		pubSubService       p2p.P2PPubSubService
		chanChainHead       chan transaction.ChainHeadEvent
		chanReqReset        chan *txPoolResetRequest
		chanReqPromote      chan *accountSet
		chanReorgDone       chan chan struct{}
		chanReorgShutdown   chan struct{}
		chanInitDone        chan struct{}
		chanQueueTxEvent    chan *transaction.Transaction
		chanRmTxs           chan []string
		query               TransactionPoolServant
		locals              *accountSet
		pending             *pendingTxs
		queue               *queueTxs
		allTxsForLook       *txLookup
		sortedByPriced      *txPricedList
		ActivationIntervals map[string]time.Time
		curState            StatePoolDB
		pendingNonces       uint64
		curMaxGasLimit      uint64
		log                 log.Logger
		level               tplogcmm.LogLevel
		network             network.Network
		handler             TransactionPoolHandler
		marshaler           codec.Marshaler
		hasher              tpcmm.Hasher
		changesSinceReorg   int
		wg                  sync.WaitGroup
	}
	type args struct {
		tx    *transaction.Transaction
		local bool
	}
	tests := []struct {
		name    string
		fields  fields
		args    args
		wantErr bool
	}{
		// TODO: Add test cases.
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			pool := &transactionPool{
				config:              tt.fields.config,
				pubSubService:       tt.fields.pubSubService,
				chanChainHead:       tt.fields.chanChainHead,
				chanReqReset:        tt.fields.chanReqReset,
				chanReqPromote:      tt.fields.chanReqPromote,
				chanReorgDone:       tt.fields.chanReorgDone,
				chanReorgShutdown:   tt.fields.chanReorgShutdown,
				chanInitDone:        tt.fields.chanInitDone,
				chanQueueTxEvent:    tt.fields.chanQueueTxEvent,
				chanRmTxs:           tt.fields.chanRmTxs,
				query:               tt.fields.query,
				locals:              tt.fields.locals,
				pending:             tt.fields.pending,
				queue:               tt.fields.queue,
				allTxsForLook:       tt.fields.allTxsForLook,
				sortedByPriced:      tt.fields.sortedByPriced,
				ActivationIntervals: tt.fields.ActivationIntervals,
				curState:            tt.fields.curState,
				pendingNonces:       tt.fields.pendingNonces,
				curMaxGasLimit:      tt.fields.curMaxGasLimit,
				log:                 tt.fields.log,
				level:               tt.fields.level,
				network:             tt.fields.network,
				handler:             tt.fields.handler,
				marshaler:           tt.fields.marshaler,
				hasher:              tt.fields.hasher,
				changesSinceReorg:   tt.fields.changesSinceReorg,
				wg:                  tt.fields.wg,
			}
			if err := pool.AddTx(tt.args.tx, tt.args.local); (err != nil) != tt.wantErr {
				t.Errorf("AddTx() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}
