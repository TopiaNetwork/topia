package txpoolinterface

import (
	"github.com/AsynkronIT/protoactor-go/actor"
	tpconfig "github.com/TopiaNetwork/topia/configuration"

	tpcrtypes "github.com/TopiaNetwork/topia/crypt/types"
	tpnet "github.com/TopiaNetwork/topia/network"
	txbasic "github.com/TopiaNetwork/topia/transaction/basic"
)

const MOD_NAME = "TransactionPool"

type TxState byte

const (
	TxState_Unknown TxState = iota
	TxState_Added
	TxState_Packed
	TxState_Published
	TxState_Republished
	TxState_DroppedForTxPoolFull
	TxState_Removed
)

type TransactionPool interface {
	AddTx(tx *txbasic.Transaction, isLocal bool) error

	RemoveTxByKey(key txbasic.TxID) error

	UpdateTx(tx *txbasic.Transaction, oldTxID txbasic.TxID) error

	PendingOfAddress(addr tpcrtypes.Address) ([]*txbasic.Transaction, error)

	PickTxs() []*txbasic.Transaction

	GetLocalTxs() []*txbasic.Transaction

	GetRemoteTxs() []*txbasic.Transaction

	Get(txID txbasic.TxID) *txbasic.Transaction

	Count() int64

	Size() int64

	Start(sysActor *actor.ActorSystem, network tpnet.Network) error

	Stop()

	SetTxPoolConfig(conf *tpconfig.TransactionPoolConfig)

	PeekTxState(hash txbasic.TxID) TxState
}
