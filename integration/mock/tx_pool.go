package mock

import (
	"context"
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
	txpool "github.com/TopiaNetwork/topia/transaction_pool"
)

type TransactionPoolMock struct {
	log          tplog.Logger
	nodeID       string
	cryptService tpcrt.CryptService
	sync         sync.RWMutex
	pendingTxs   []txbasic.Transaction //tx hex hash -> Transaction
}

func NewTransactionPoolMock(log tplog.Logger, nodeID string, cryptService tpcrt.CryptService) *TransactionPoolMock {
	fromPriKey, _, _ := cryptService.GeneratePriPubKey()

	txPool := &TransactionPoolMock{
		log:          log,
		nodeID:       nodeID,
		cryptService: cryptService,
	}

	for i := 0; i < 5; i++ {
		_, toPubKey, _ := cryptService.GeneratePriPubKey()
		toAddr, _ := cryptService.CreateAddress(toPubKey)

		tx := txuni.ConstructTransactionWithUniversalTransfer(log, cryptService, fromPriKey, fromPriKey, uint64(i+1), 200, 500, toAddr,
			[]txuni.TargetItem{{currency.TokenSymbol_Native, big.NewInt(10)}})

		txPool.pendingTxs = append(txPool.pendingTxs, *tx)
	}

	return txPool
}

func (txm *TransactionPoolMock) AddTx(tx *txbasic.Transaction) error {
	//TODO implement me
	panic("implement me")
}

func (txm *TransactionPoolMock) RemoveTxByKey(key txpool.TxKey) error {
	txm.sync.Lock()
	defer txm.sync.Unlock()

	for i, tx := range txm.pendingTxs {
		txHash, _ := tx.HashHex()
		if txHash == string(key) {
			txm.pendingTxs = append(txm.pendingTxs[:i], txm.pendingTxs[i+1:]...)
			i--
		}
	}

	return nil
}

func (txm *TransactionPoolMock) Reset() error {
	//TODO implement me
	panic("implement me")
}

func (txm *TransactionPoolMock) UpdateTx(tx *txbasic.Transaction) error {
	//TODO implement me
	panic("implement me")
}

func (txm *TransactionPoolMock) Pending() ([]txbasic.Transaction, error) {
	txm.sync.Lock()
	defer txm.sync.Unlock()

	newTxs := make([]txbasic.Transaction, len(txm.pendingTxs))

	copy(newTxs, txm.pendingTxs)

	return newTxs, nil
}

func (txm *TransactionPoolMock) Size() int {
	//TODO implement me
	panic("implement me")
}

func (txm *TransactionPoolMock) processBlockAddedEvent(ctx context.Context, data interface{}) error {
	marshaler := codec.CreateMarshaler(codec.CodecType_PROTO)
	if block, ok := data.(*tpchaintypes.Block); ok {
		for _, txBytes := range block.Data.Txs {
			var tx txbasic.Transaction
			marshaler.Unmarshal(txBytes, &tx)
			txHash, _ := tx.HashHex()
			txm.RemoveTxByKey(txpool.TxKey(txHash))
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

						txm.pendingTxs = append(txm.pendingTxs, *tx)
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
