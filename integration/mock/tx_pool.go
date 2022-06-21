package mock

import (
	"context"
	tpcrtypes "github.com/TopiaNetwork/topia/crypt/types"
	"math/big"
	"sync"
	"time"

	"github.com/AsynkronIT/protoactor-go/actor"

	tpchaintypes "github.com/TopiaNetwork/topia/chain/types"
	"github.com/TopiaNetwork/topia/codec"
	tpcrt "github.com/TopiaNetwork/topia/crypt"
	"github.com/TopiaNetwork/topia/currency"
	"github.com/TopiaNetwork/topia/eventhub"
	tplog "github.com/TopiaNetwork/topia/log"
	tpnet "github.com/TopiaNetwork/topia/network"
	txbasic "github.com/TopiaNetwork/topia/transaction/basic"
	txuni "github.com/TopiaNetwork/topia/transaction/universal"
	txpooli "github.com/TopiaNetwork/topia/transaction_pool/interface"
)

type TransactionPoolMock struct {
	log          tplog.Logger
	nodeID       string
	cryptService tpcrt.CryptService
	sync         sync.RWMutex
	pendingTxs   map[txbasic.TxID]*txbasic.Transaction //tx hex hash -> Transaction
}

func (txm *TransactionPoolMock) PendingOfAddress(addr tpcrtypes.Address) ([]*txbasic.Transaction, error) {
	//TODO implement me
	panic("implement me")
}

func (txm *TransactionPoolMock) Count() int64 {
	//TODO implement me
	panic("implement me")
}

func (txm *TransactionPoolMock) Size() int64 {
	//TODO implement me
	panic("implement me")
}

func (txm *TransactionPoolMock) RemoveTxHashs(hashs []txbasic.TxID) []error {
	//TODO implement me
	panic("implement me")
}

func (txm *TransactionPoolMock) UpdateTx(tx *txbasic.Transaction, txKey txbasic.TxID) error {
	//TODO implement me
	panic("implement me")
}

func (txm *TransactionPoolMock) TruncateTxPool() {
	//TODO implement me
	panic("implement me")
}

func (txm *TransactionPoolMock) SetTxPoolConfig(conf txpooli.TransactionPoolConfig) {
	//TODO implement me
	panic("implement me")
}

func (txm *TransactionPoolMock) PeekTxState(hash txbasic.TxID) txpooli.TransactionState {
	//TODO implement me
	panic("implement me")
}

func NewTransactionPoolMock(log tplog.Logger, nodeID string, cryptService tpcrt.CryptService) *TransactionPoolMock {
	fromPriKey, _, _ := cryptService.GeneratePriPubKey()

	txPool := &TransactionPoolMock{
		log:          log,
		nodeID:       nodeID,
		cryptService: cryptService,
		pendingTxs:   make(map[txbasic.TxID]*txbasic.Transaction),
	}

	for i := 0; i < 5; i++ {
		_, toPubKey, _ := cryptService.GeneratePriPubKey()
		toAddr, _ := cryptService.CreateAddress(toPubKey)

		tx := txuni.ConstructTransactionWithUniversalTransfer(log, cryptService, fromPriKey, fromPriKey, uint64(i+1), 200, 500, toAddr,
			[]txuni.TargetItem{{currency.TokenSymbol_Native, big.NewInt(10)}})

		txID, _ := tx.TxID()

		txPool.pendingTxs[txID] = tx
	}

	return txPool
}

func (txm *TransactionPoolMock) AddTx(tx *txbasic.Transaction, local bool) error {
	//TODO implement me
	panic("implement me")
}

func (txm *TransactionPoolMock) RemoveTxByKey(key txbasic.TxID) error {
	txm.sync.Lock()
	defer txm.sync.Unlock()

	if _, ok := txm.pendingTxs[key]; ok {
		delete(txm.pendingTxs, key)
	}

	return nil
}

func (txm *TransactionPoolMock) PickTxs() []*txbasic.Transaction {
	txm.sync.Lock()
	defer txm.sync.Unlock()

	newTxs := make([]*txbasic.Transaction, len(txm.pendingTxs))

	i := 0
	for _, tx := range txm.pendingTxs {
		newTxs[i] = tx
		i++
	}

	return newTxs
}

func (txm *TransactionPoolMock) processBlockAddedEvent(ctx context.Context, data interface{}) error {
	marshaler := codec.CreateMarshaler(codec.CodecType_PROTO)
	if block, ok := data.(*tpchaintypes.Block); ok {
		for _, txBytes := range block.Data.Txs {
			var tx txbasic.Transaction
			marshaler.Unmarshal(txBytes, &tx)
			txID, _ := tx.TxID()
			txm.RemoveTxByKey(txID)
		}

		return nil
	}

	panic("Unknown sub event data")
}

func (txm *TransactionPoolMock) produceTxsTimer(ctx context.Context) {
	go func() {
		timer := time.NewTicker(500 * time.Millisecond)
		defer timer.Stop()
		for {
			select {
			case <-timer.C:
				err := func() error {
					txm.sync.Lock()
					defer txm.sync.Unlock()
					fromPriKey, _, _ := txm.cryptService.GeneratePriPubKey()
					for i := 0; i < 5; i++ {
						_, toPubKey, _ := txm.cryptService.GeneratePriPubKey()
						toAddr, _ := txm.cryptService.CreateAddress(toPubKey)

						tx := txuni.ConstructTransactionWithUniversalTransfer(txm.log, txm.cryptService, fromPriKey, fromPriKey, uint64(i+1), 200, 500, toAddr,
							[]txuni.TargetItem{{currency.TokenSymbol_Native, big.NewInt(10)}})

						txID, _ := tx.TxID()

						txm.pendingTxs[txID] = tx
					}

					return nil
				}()
				if err != nil {
					continue
				}
			case <-ctx.Done():
				return
			}
		}

	}()
}

func (txm *TransactionPoolMock) Start(sysActor *actor.ActorSystem, network tpnet.Network) error {
	ctx := context.Background()

	txm.produceTxsTimer(ctx)
	eventhub.GetEventHubManager().GetEventHub(txm.nodeID).Observe(ctx, eventhub.EventName_BlockAdded, txm.processBlockAddedEvent)
	return nil
}

func (txm *TransactionPoolMock) SysShutDown() {
	//TODO implement me
	panic("implement me")
}
