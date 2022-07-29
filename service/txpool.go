package service

import (
	"github.com/AsynkronIT/protoactor-go/actor"
	tpcrtypes "github.com/TopiaNetwork/topia/crypt/types"
	tpnet "github.com/TopiaNetwork/topia/network"
	txbasic "github.com/TopiaNetwork/topia/transaction/basic"
	txpooli "github.com/TopiaNetwork/topia/transaction_pool/interface"
)

type TxPoolService interface {
	AddTx(tx *txbasic.Transaction, local bool) error

	RemoveTxByKey(key txbasic.TxID) error

	RemoveTxHashes(hashes []txbasic.TxID) []error

	UpdateTx(tx *txbasic.Transaction, txKey txbasic.TxID) error

	PendingOfAddress(addr tpcrtypes.Address) ([]*txbasic.Transaction, error)

	PickTxs() []*txbasic.Transaction

	Count() int64

	Size() int64

	TruncateTxPool()

	Start(sysActor *actor.ActorSystem, network tpnet.Network) error

	SysShutDown()

	SetTxPoolConfig(conf txpooli.TransactionPoolConfig)

	PeekTxState(hash txbasic.TxID) txpooli.TransactionState
}

type txPoolService struct {
	txpooli.TransactionPool
}

func NewTxPoolService(txPool txpooli.TransactionPool) TxPoolService {
	return &txPoolService{
		TransactionPool: txPool,
	}
}
