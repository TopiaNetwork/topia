package txpoolinterface

import (
	"github.com/AsynkronIT/protoactor-go/actor"
	tpconfig "github.com/TopiaNetwork/topia/configuration"

	tpcrtypes "github.com/TopiaNetwork/topia/crypt/types"
	tpnet "github.com/TopiaNetwork/topia/network"
	txbasic "github.com/TopiaNetwork/topia/transaction/basic"
)

const MOD_NAME = "TransactionPool"

type TransactionState string

const (
	StateTxAdded              TransactionState = "transaction Added"
	StateTxRepublished                         = "transaction republished"
	StateTxNil                                 = "no transaction state for this tx "
	StateDroppedForTxPoolFull                  = "transaction dropped for txPool is full"
)

type TransactionPool interface {
	AddTx(tx *txbasic.Transaction, isLocal bool) error

	RemoveTxByKey(key txbasic.TxID) error

	UpdateTx(tx *txbasic.Transaction, txKey txbasic.TxID) error

	PendingOfAddress(addr tpcrtypes.Address) ([]*txbasic.Transaction, error)

	PickTxs() []*txbasic.Transaction

	Count() int64

	Size() int64

	TruncateTxPool()

	Start(sysActor *actor.ActorSystem, network tpnet.Network) error

	SysShutDown()

	SetTxPoolConfig(conf *tpconfig.TransactionPoolConfig)

	PeekTxState(hash txbasic.TxID) TransactionState
}
