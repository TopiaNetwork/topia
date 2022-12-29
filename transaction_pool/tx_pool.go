package transactionpool

import (
	"context"
	"sync"
	"sync/atomic"
	"time"

	"github.com/AsynkronIT/protoactor-go/actor"
	"github.com/hashicorp/golang-lru"

	tpchaintypes "github.com/TopiaNetwork/topia/chain/types"
	"github.com/TopiaNetwork/topia/codec"
	tpcmm "github.com/TopiaNetwork/topia/common"
	tpconfig "github.com/TopiaNetwork/topia/configuration"
	tpcrtypes "github.com/TopiaNetwork/topia/crypt/types"
	"github.com/TopiaNetwork/topia/eventhub"
	"github.com/TopiaNetwork/topia/ledger"
	tplog "github.com/TopiaNetwork/topia/log"
	tplogcmm "github.com/TopiaNetwork/topia/log/common"
	tpnet "github.com/TopiaNetwork/topia/network"
	tpnetprotoc "github.com/TopiaNetwork/topia/network/protocol"
	"github.com/TopiaNetwork/topia/service"
	txbasic "github.com/TopiaNetwork/topia/transaction/basic"
	txpoolcore "github.com/TopiaNetwork/topia/transaction_pool/core"
	txpooli "github.com/TopiaNetwork/topia/transaction_pool/interface"
)

type transactionPool struct {
	exeDomainID   string
	nodeID        string
	log           tplog.Logger
	level         tplogcmm.LogLevel
	ctx           context.Context
	marshaler     codec.Marshaler
	hasher        tpcmm.Hasher
	config        *tpconfig.TransactionPoolConfig
	txServant     TransactionPoolServant
	txMsgSub      TxMsgSubProcessor
	handler       TransactionPoolHandler
	mu            sync.RWMutex
	txSizeBytes   int64
	txCount       int64
	txCache       *lru.Cache
	txsCollect    *txpoolcore.CollectTxs
	localAddrs    map[tpcrtypes.Address]struct{}
	blockRevertCh chan []*tpchaintypes.Block
}

func NewTransactionPool(exeDomainID string,
	nodeID string,
	ctx context.Context,
	conf *tpconfig.TransactionPoolConfig,
	level tplogcmm.LogLevel,
	log tplog.Logger,
	codecType codec.CodecType,
	stateQueryService service.StateQueryService,
	blockService service.BlockService,
	network tpnet.Network,
	ledger ledger.Ledger) txpooli.TransactionPool {
	confNew := conf.Check()
	poolLog := tplog.CreateModuleLogger(level, "TransactionPool", log)

	pool := &transactionPool{
		exeDomainID:   exeDomainID,
		nodeID:        nodeID,
		config:        confNew,
		log:           poolLog,
		level:         level,
		ctx:           ctx,
		marshaler:     codec.CreateMarshaler(codecType),
		hasher:        tpcmm.NewBlake2bHasher(0),
		txServant:     newTransactionPoolServant(stateQueryService, blockService, network, ledger),
		txsCollect:    txpoolcore.NewCollectTxs(),
		localAddrs:    make(map[tpcrtypes.Address]struct{}),
		blockRevertCh: make(chan []*tpchaintypes.Block),
	}

	pool.txCache, _ = lru.New(TxCacheSize)

	if pool.config.IsLoadCfg {
		pool.config, _ = pool.txServant.loadPoolConfig(pool.marshaler)
	}
	if pool.config.IsLoadTxs {
		pool.txServant.loadAllTxs(pool.marshaler, pool.addTx)
	}

	txMsgSub := NewTxMsgSubProcessor(exeDomainID, nodeID, poolLog, pool)
	pool.txMsgSub = txMsgSub

	poolHandler := NewTransactionPoolHandler(poolLog, pool, txMsgSub)
	pool.handler = poolHandler

	pool.txServant.CreateTopic(tpnetprotoc.AsyncSendProtocolID + "/" + pool.exeDomainID)

	return pool
}

func (pool *transactionPool) Dispatch(context actor.Context, data []byte) {
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
		err = pool.processTx(&msg)
		if err != nil {
			return
		}
	default:
		pool.log.Errorf("TransactionPool receive invalid msg %d", txPoolMsg.MsgType)
		return
	}
}

func (pool *transactionPool) processTx(msg *TxMessage) error {
	err := pool.handler.ProcessTx(pool.ctx, msg)
	if err != nil {
		return err

	}
	return nil
}
func (pool *transactionPool) AddTx(tx *txbasic.Transaction, isLocal bool) error {
	if isLocal {
		return pool.addLocal(tx)
	} else {
		return pool.addRemote(tx)
	}
}

func (pool *transactionPool) addLocal(tx *txbasic.Transaction) error {
	fromAddr := tpcrtypes.NewFromBytes(tx.Head.FromAddr)
	pool.localAddrs[fromAddr] = struct{}{}

	wTx, err := pool.addTx(tx)
	if err == nil {
		pool.txServant.saveTxIntoStore(pool.marshaler, wTx.TxID(), tx)
		/*err = pool.txServant.PublishTx(pool.ctx, pool.marshaler, tpnetprotoc.AsyncSendProtocolID+"/"+pool.exeDomainID, pool.exeDomainID, pool.nodeID, wTx.OriginTx())
		if err == nil {
			wTx.UpdateState(txpooli.TxState_Published)
		}*/
	}

	return err
}

func (pool *transactionPool) addRemote(tx *txbasic.Transaction) error {
	_, err := pool.addTx(tx)
	return err
}

func (pool *transactionPool) addTx(tx *txbasic.Transaction) (txpoolcore.TxWrapper, error) {
	txId, _ := tx.TxID()

	if pool.Count()+1 > pool.config.TxPoolMaxCnt {
		pool.txCache.Add(txId, txpooli.TxState_DroppedForTxPoolFull)
		return nil, ErrTxPoolFull
	}

	if pool.Size()+int64(tx.Size()) > pool.config.TxPoolMaxSize {
		pool.txCache.Add(txId, txpooli.TxState_DroppedForTxPoolFull)
		return nil, ErrTxPoolFull
	}

	curHeight, err := pool.txServant.CurrentHeight()
	if err != nil {
		return nil, err
	}
	wTx, err := txpoolcore.GenerateTxWrapper(tx, curHeight)
	if err != nil {
		return nil, err
	}
	err = pool.txsCollect.AddTx(wTx)
	if err == nil {
		pool.txCache.Add(txId, txpooli.TxState_Added)
		atomic.AddInt64(&pool.txCount, 1)
		atomic.AddInt64(&pool.txSizeBytes, int64(wTx.Size()))
		return wTx, nil
	} else {
		return nil, err
	}
}

func (pool *transactionPool) UpdateTx(tx *txbasic.Transaction, oldTxID txbasic.TxID) error {
	wTxOld := pool.txsCollect.GetTx(oldTxID)
	if wTxOld == nil {
		return ErrTxNotExist
	}

	curHeight, err := pool.txServant.CurrentHeight()
	if err != nil {
		return err
	}

	wTx, err := txpoolcore.GenerateTxWrapper(tx, curHeight)
	if err != nil {
		return err
	}

	if wTxOld.CanReplace(wTx) {
		_, err := pool.addTx(tx)
		if err != nil {
			return err
		}

		pool.txServant.removeTxFromStore(oldTxID)
		pool.txServant.saveTxIntoStore(pool.marshaler, wTx.TxID(), tx)
		wTx.UpdateState(txpooli.TxState_Added)

		//commit when unit testing
		//**********
		//txRemoved := &eventhub.TxPoolEvent{
		//	EvType: eventhub.TxPoolEVType_Removed,
		//	Tx:     txOldWrapped.Tx,
		//}
		//eventhub.GetEventHubManager().GetEventHub(pool.nodeId).Trig(pool.ctx, eventhub.EventName_TxPoolChanged, txRemoved)
		////commit when unit testing
		//txAdded := &eventhub.TxPoolEvent{
		//	EvType: eventhub.TxPoolEVType_Received,
		//	Tx:     txOldWrapped.Tx,
		//}
		//eventhub.GetEventHubManager().GetEventHub(pool.nodeId).Trig(pool.ctx, eventhub.EventName_TxPoolChanged, txAdded)
		//***********
	}
	return nil
}

func (pool *transactionPool) RemoveTxByKey(txID txbasic.TxID) error {
	wTx := pool.txsCollect.RemoveTx(txID)
	if wTx != nil {
		pool.txCache.Remove(txID)
		atomic.AddInt64(&pool.txCount, -1)
		atomic.AddInt64(&pool.txSizeBytes, -int64(wTx.Size()))
		pool.txServant.removeTxFromStore(txID)
	}

	return nil
}

func (pool *transactionPool) Get(txID txbasic.TxID) *txbasic.Transaction {
	return pool.txsCollect.GetTx(txID).OriginTx()
}

func (pool *transactionPool) Size() int64 {
	return atomic.LoadInt64(&pool.txSizeBytes)
}
func (pool *transactionPool) PendingAccountTxCnt(address tpcrtypes.Address) int64 {
	txs, _ := pool.PendingOfAddress(address)
	return int64(len(txs))
}

func (pool *transactionPool) Count() int64 {
	return atomic.LoadInt64(&pool.txCount)
}

func (pool *transactionPool) Start(sysActor *actor.ActorSystem, network tpnet.Network) error {
	actorPID, err := CreateTransactionPoolActor(pool.level, pool.log, sysActor, pool)
	if err != nil {
		pool.log.Panicf("CreateTransactionPoolActor error: %v", err)
		return err
	}
	network.RegisterModule(txpooli.MOD_NAME, actorPID, pool.marshaler)

	pool.txServant.Subscribe(pool.ctx, tpnetprotoc.AsyncSendProtocolID+"/"+pool.exeDomainID,
		true,
		pool.txMsgSub.Validate)

	ObsID, err = eventhub.GetEventHubManager().GetEventHub(pool.nodeID).
		Observe(pool.ctx, eventhub.EventName_BlockAdded, pool.handler.processBlockAddedEvent)
	if err != nil {
		pool.log.Panicf("processBlockAddedEvent error:%s", err)
	}

	pool.log.Infof("processBlockAddedEvent,obsID:%s", ObsID)

	pool.txsCollect.Start(pool.ctx)

	return nil
}

func (pool *transactionPool) Stop() {
	pool.txServant.UnSubscribe(tpnetprotoc.SyncProtocolID_Msg)
	eventhub.GetEventHubManager().GetEventHub(pool.nodeID).UnObserve(pool.ctx, ObsID, eventhub.EventName_BlockAdded)
	pool.txServant.savePoolConfig(pool.config, pool.marshaler)

	pool.log.Info("TransactionPool stopped")
}

func (pool *transactionPool) PickTxs() []*txbasic.Transaction {
	defer func(t0 time.Time) {
		pool.log.Infof("PickTxs cost time: %d", time.Since(t0))
	}(time.Now())

	return pool.txsCollect.PickTxs(pool.config.BlockMaxBytes, pool.config.BlockMaxGas)
}
func (pool *transactionPool) GetLocalTxs() []*txbasic.Transaction {
	return pool.txsCollect.AllOfAddressFunc(func(addr tpcrtypes.Address) bool {
		if _, ok := pool.localAddrs[addr]; ok {
			return true
		}

		return false
	})
}
func (pool *transactionPool) GetRemoteTxs() []*txbasic.Transaction {
	return pool.txsCollect.AllOfAddressFunc(func(addr tpcrtypes.Address) bool {
		if _, ok := pool.localAddrs[addr]; !ok {
			return true
		}

		return false
	})
}

func (pool *transactionPool) PendingOfAddress(addr tpcrtypes.Address) ([]*txbasic.Transaction, error) {
	return pool.txsCollect.PendingOfAddress(addr)
}

func (pool *transactionPool) PeekTxState(txid txbasic.TxID) txpooli.TxState {
	value, ok := pool.txCache.Peek(txid)
	if ok {
		return value.(txpooli.TxState)
	} else {
		return txpooli.TxState_Unknown
	}
}
