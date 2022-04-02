package mock

import (
	"github.com/AsynkronIT/protoactor-go/actor"
	tpcrt "github.com/TopiaNetwork/topia/crypt"
	tplog "github.com/TopiaNetwork/topia/log"
	"sync"

	tpnet "github.com/TopiaNetwork/topia/network"
	txbasic "github.com/TopiaNetwork/topia/transaction/basic"
	txpool "github.com/TopiaNetwork/topia/transaction_pool"
)

type TransactionPoolMock struct {
	log          tplog.Logger
	cryptService tpcrt.CryptService
	sync         sync.RWMutex
	pendingTxs   map[string]txbasic.Transaction //tx hex hash -> Transaction
}

func NewTransactionPoolMock(log tplog.Logger, cryptService tpcrt.CryptService) *TransactionPoolMock {
	/*priKey, pubKey, _ := cryptService.GeneratePriPubKey()
	fromAddr, _ := cryptService.CreateAddress(pubKey)
	for i := 0; i < 5; i++ {
		tAddrT, _ := cryptService.CreateAddress(pubKey)
	}

	return &TransactionPoolMock{
		log:          log,
		cryptService: cryptService,
		pendingTxs:   make(map[string]txbasic.Transaction),
	}

	*/

	return nil
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
	//TODO implement me
	panic("implement me")
}

func (txm *TransactionPoolMock) Size() int {
	//TODO implement me
	panic("implement me")
}

func (txm *TransactionPoolMock) Start(sysActor *actor.ActorSystem, network tpnet.Network) error {
	//TODO implement me
	panic("implement me")
}
