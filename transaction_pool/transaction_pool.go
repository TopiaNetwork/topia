package transactionpool

import (
	tx "github.com/TopiaNetwork/topia/transaction/basic"
	"sync"

	"github.com/AsynkronIT/protoactor-go/actor"

	"github.com/TopiaNetwork/topia/codec"
	tplog "github.com/TopiaNetwork/topia/log"
	tplogcmm "github.com/TopiaNetwork/topia/log/common"
	tpnet "github.com/TopiaNetwork/topia/network"
)

type TxKey string

type TransactionPool interface {
	AddTx(tx *tx.Transaction) error

	RemoveTxByKey(key TxKey) error

	Reset() error

	UpdateTx(tx *tx.Transaction) error

	Pending() ([]tx.Transaction, error)

	Size() int

	Start(sysActor *actor.ActorSystem, network tpnet.Network) error
}

type transactionPool struct {
	txLock    sync.Mutex
	log       tplog.Logger
	level     tplogcmm.LogLevel
	network   tpnet.Network
	handler   TransactionPoolHandler
	marshaler codec.Marshaler
}

func NewTransactionPool(level tplogcmm.LogLevel, log tplog.Logger, codecType codec.CodecType) TransactionPool {
	poolLog := tplog.CreateModuleLogger(level, "TransactionPool", log)
	return &transactionPool{
		log:       poolLog,
		level:     level,
		handler:   NewTransactionPoolHandler(poolLog),
		marshaler: codec.CreateMarshaler(codecType),
	}
}

func (pool *transactionPool) AddTx(tx *tx.Transaction) error {
	panic("implement me")
}

func (pool *transactionPool) RemoveTxByKey(key TxKey) error {
	panic("implement me")
}

func (pool *transactionPool) Reset() error {
	panic("implement me")
}

func (pool *transactionPool) UpdateTx(tx *tx.Transaction) error {
	panic("implement me")
}

func (pool *transactionPool) Pending() ([]tx.Transaction, error) {
	panic("implement me")
}

func (pool *transactionPool) Size() int {
	panic("implement me")
}

func (pool *transactionPool) processTx(msg *TxMessage) error {
	return pool.handler.ProcessTx(msg)
}

func (pool *transactionPool) dispatch(context actor.Context, data []byte) {
	var txPoolMsg TxPoolMessage
	err := pool.marshaler.Unmarshal(data, &txPoolMsg)
	if err != nil {
		pool.log.Errorf("TransactionPool receive invalid data %v", data)
		return
	}

	switch txPoolMsg.MsgType {
	case TxPoolMessage_Tx:
		var msg TxMessage
		err := pool.marshaler.Unmarshal(txPoolMsg.Data, &msg)
		if err != nil {
			pool.log.Errorf("TransactionPool unmarshal msg %d err %v", txPoolMsg.MsgType, err)
			return
		}
		pool.processTx(&msg)
	default:
		pool.log.Errorf("TransactionPool receive invalid msg %d", txPoolMsg.MsgType)
		return
	}
}

func (pool *transactionPool) Start(sysActor *actor.ActorSystem, network tpnet.Network) error {
	actorPID, err := CreateTransactionPoolActor(pool.level, pool.log, sysActor, pool)
	if err != nil {
		pool.log.Panicf("CreateTransactionPoolActor error: %v", err)
		return err
	}

	network.RegisterModule("TransactionPool", actorPID, pool.marshaler)

	return nil
}
