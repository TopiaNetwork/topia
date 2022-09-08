package core

import (
	"time"

	tpcrtypes "github.com/TopiaNetwork/topia/crypt/types"
	txbasic "github.com/TopiaNetwork/topia/transaction/basic"
	txuni "github.com/TopiaNetwork/topia/transaction/universal"
	txpooli "github.com/TopiaNetwork/topia/transaction_pool/interface"
)

type TxWrapperUni struct {
	txID      txbasic.TxID
	category  txbasic.TransactionCategory
	txState   txpooli.TxState
	size      int
	timeStamp time.Time
	height    uint64
	fromAddr  tpcrtypes.Address
	nonce     uint64
	gasPrice  uint64
	fromPeers map[string]struct{}
	orginTx   *txbasic.Transaction
}

func GenerateTxWrapperUni(tx *txbasic.Transaction, height uint64) (*TxWrapperUni, error) {
	txID, _ := tx.TxID()
	category := txbasic.TransactionCategory(tx.Head.Category)
	fromAddr := tpcrtypes.NewFromBytes(tx.Head.FromAddr)
	nonce := tx.Head.Nonce

	var txUni txuni.TransactionUniversal
	if err := txUni.Unmarshal(tx.Data.Specification); err != nil {
		return nil, err
	}
	gasPrice := txUni.Head.GasPrice

	return &TxWrapperUni{
		txID:      txID,
		category:  category,
		txState:   txpooli.TxState_Unknown,
		size:      tx.Size(),
		timeStamp: time.Now().UTC(),
		height:    height,
		fromAddr:  fromAddr,
		nonce:     nonce,
		gasPrice:  gasPrice,
		fromPeers: make(map[string]struct{}),
		orginTx:   tx,
	}, nil
}

func (tw *TxWrapperUni) TxID() txbasic.TxID {
	return tw.txID
}

func (tw *TxWrapperUni) Category() txbasic.TransactionCategory {
	return tw.category
}

func (tw *TxWrapperUni) UpdateState(s txpooli.TxState) {
	tw.txState = s
}

func (tw *TxWrapperUni) TxState() txpooli.TxState {
	return tw.txState
}

func (tw *TxWrapperUni) Size() int {
	return tw.size
}

func (tw *TxWrapperUni) TimeStamp() time.Time {
	return tw.timeStamp
}

func (tw *TxWrapperUni) Height() uint64 {
	return tw.height
}

func (tw *TxWrapperUni) FromAddr() tpcrtypes.Address {
	return tw.fromAddr
}

func (tw *TxWrapperUni) Nonce() uint64 {
	return tw.nonce
}

func (tw *TxWrapperUni) AddFromPeer(peerID string) bool {
	if _, ok := tw.fromPeers[peerID]; ok {
		return false
	}

	tw.fromPeers[peerID] = struct{}{}

	return true
}

func (tw *TxWrapperUni) HasFromPeer(peerID string) bool {
	if _, ok := tw.fromPeers[peerID]; ok {
		return false
	}

	tw.fromPeers[peerID] = struct{}{}

	return true
}

func (tw *TxWrapperUni) OriginTx() *txbasic.Transaction {
	return tw.orginTx
}

func (tw *TxWrapperUni) Less(sWay SortWay, target TxWrapper) bool {
	targetUni, ok := target.(*TxWrapperUni)
	if !ok {
		return false
	}
	lessMisc := func() bool {
		if tw.gasPrice > targetUni.gasPrice {
			return true
		} else if tw.gasPrice == targetUni.gasPrice && tw.height < targetUni.height {
			return true
		} else if tw.height == targetUni.height && tw.timeStamp.Before(targetUni.timeStamp) {
			return true
		}

		return false
	}

	lessHeight := func() bool {
		return tw.height < targetUni.height
	}

	lessTime := func() bool {
		return tw.timeStamp.Before(targetUni.timeStamp)
	}

	switch sWay {
	case SortWay_Misc:
		return lessMisc()
	case SortWay_Height:
		return lessHeight()
	case SortWay_Time:
		return lessTime()
	default:
		panic("Invalid sort tx type")
	}

	return false
}

func (tw *TxWrapperUni) CanReplace(other TxWrapper) bool {
	otherUni, ok := other.(*TxWrapperUni)
	if !ok {
		return false
	}

	return tw.txState != txpooli.TxState_Packed && otherUni.fromAddr == tw.fromAddr && otherUni.nonce == tw.nonce && otherUni.gasPrice > tw.gasPrice

}
