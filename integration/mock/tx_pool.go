package mock

import (
	"github.com/AsynkronIT/protoactor-go/actor"
	"github.com/TopiaNetwork/topia/chain"
	tpcrt "github.com/TopiaNetwork/topia/crypt"
	tplog "github.com/TopiaNetwork/topia/log"
	"math/big"
	"sync"

	tpnet "github.com/TopiaNetwork/topia/network"
	txbasic "github.com/TopiaNetwork/topia/transaction/basic"
	txuni "github.com/TopiaNetwork/topia/transaction/universal"
	txpool "github.com/TopiaNetwork/topia/transaction_pool"
)

type TransactionPoolMock struct {
	log          tplog.Logger
	cryptService tpcrt.CryptService
	sync         sync.RWMutex
	pendingTxs   []txbasic.Transaction //tx hex hash -> Transaction
}

func NewTransactionPoolMock(log tplog.Logger, cryptService tpcrt.CryptService) *TransactionPoolMock {
	fromPriKey, _, _ := cryptService.GeneratePriPubKey()

	txPool := &TransactionPoolMock{
		log:          log,
		cryptService: cryptService,
	}

	for i := 0; i < 5; i++ {
		_, toPubKey, _ := cryptService.GeneratePriPubKey()
		toAddr, _ := cryptService.CreateAddress(toPubKey)

		tx := txuni.ConstructTransactionWithUniversalTransfer(log, cryptService, fromPriKey, fromPriKey, 1, 200, 500, toAddr,
			[]txuni.TargetItem{{chain.TokenSymbol_Native, big.NewInt(10)}})

		txPool.pendingTxs = append(txPool.pendingTxs, *tx)
	}

	return txPool
}

func (txm *TransactionPoolMock) AddTx(tx *txbasic.Transaction) error {
	//TODO implement me
	panic("implement me")
}

func (txm *TransactionPoolMock) RemoveTxByKey(key txpool.TxKey) error {
	//TODO implement me
	panic("implement me")
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
	return txm.pendingTxs, nil
}

func (txm *TransactionPoolMock) Size() int {
	//TODO implement me
	panic("implement me")
}

func (txm *TransactionPoolMock) Start(sysActor *actor.ActorSystem, network tpnet.Network) error {
	//TODO implement me
	panic("implement me")
}
