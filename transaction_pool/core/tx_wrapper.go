package core

import (
	"time"

	tpcrtypes "github.com/TopiaNetwork/topia/crypt/types"
	txbasic "github.com/TopiaNetwork/topia/transaction/basic"
	txpooli "github.com/TopiaNetwork/topia/transaction_pool/interface"
)

type TxWrapper interface {
	TxID() txbasic.TxID

	Category() txbasic.TransactionCategory

	UpdateState(s txpooli.TxState)

	TxState() txpooli.TxState

	Size() int

	TimeStamp() time.Time

	Height() uint64

	FromAddr() tpcrtypes.Address

	Nonce() uint64

	AddFromPeer(peerID string) bool

	HasFromPeer(peerID string) bool

	OriginTx() *txbasic.Transaction

	Less(sWay SortWay, target TxWrapper) bool

	CanReplace(other TxWrapper) bool
}

func GenerateTxWrapper(tx *txbasic.Transaction, height uint64) (TxWrapper, error) {
	category := txbasic.TransactionCategory(tx.Head.Category)
	switch category {
	case txbasic.TransactionCategory_Topia_Universal:
		return GenerateTxWrapperUni(tx, height)
	default:
		panic("Unsupported tx type: " + category)
	}
}
