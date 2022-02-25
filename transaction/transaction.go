package transaction

import (
	"container/heap"
	"encoding/hex"
	"github.com/TopiaNetwork/topia/account"
	tpcmm "github.com/TopiaNetwork/topia/common"
	"github.com/TopiaNetwork/topia/common/types"
	//"github.com/TopiaNetwork/topia/common/types"
)

// TxID is a hash value of a Transaction.
type TxID string

type TransactionType byte

type ChainHeadEvent struct{Block *types.Block}

const (
	TransactionType_Unknown TransactionType = iota
	TransactionType_Transfer
	TransactionType_ContractDeploy
	TransactionType_ContractInvoke
	TransactionType_NativeInvoke
	TransactionType_Pay
	TransactionType_Relay
	TransactionType_DataTransfer
)

func (m *Transaction) GetType() TransactionType {
	panic("implement me")
}

func (m *Transaction) TxID() (TxID, error) {
	hasher := tpcmm.NewBlake2bHasher(0)
	txBytes, err := m.Marshal()
	if err != nil {
		return "", err
	}

	hashBytes := hasher.Compute(string(txBytes))

	return TxID(hex.EncodeToString(hashBytes)), nil
}



type Item struct{
	HashStr  *Transaction
	Priority uint64
	index    int
}
type SortedTxs []*Item

func (stx SortedTxs) Len() int          { return len(stx)}
func (stx SortedTxs) Less(i,j int) bool {
	return stx[i].Priority < stx[j].Priority
}
func (stx SortedTxs) Swap(i,j int)      {
	stx[i],stx[j] = stx[j],stx[i]
	stx[i].index = i
	stx[j].index = j
}
func (stx *SortedTxs) Push(x interface{}) {
	n := len(*stx)
	item :=x.(*Item)
	item.index = n
	*stx = append(*stx,item)
}
func (stx *SortedTxs) Pop() interface{} {
	old := *stx
	n := len(*stx)
	item := old[n-1]
	item.index = n
	*stx = old[0:n-1]
	return item
}
func (stx *SortedTxs) Merge(stxb SortedTxs) interface{} {
	for _,tx := range stxb {
		var item = &Item{
			tx.HashStr,
			tx.Priority,
			tx.index,
		}
		heap.Push(stx,item)
	}
	return stx
}

type TxByPriceAndTime []*Transaction

func (s TxByPriceAndTime) Len() int { return len(s) }
func (s TxByPriceAndTime) Less(i, j int) bool {
	if s[i].GasPrice < s[j].GasPrice {return true}
	if s[i].GasPrice == s[j].GasPrice{
		return s[i].Time.Before(s[j].Time)
	}
	return false
}
func (s TxByPriceAndTime) Swap(i,j int) {s[i], s[j] = s[j], s[i]}
func (s *TxByPriceAndTime) Push(x interface{}) {
	*s = append(*s, x.(*Transaction))
}
func (s *TxByPriceAndTime) Pop() interface{} {
	old := *s
	n := len(old)
	x := old[n-1]
	*s = old[0:n-1]
	return x
}

type TxsByPriceAndNonce struct {
	txs 	map[account.Address][]*Transaction
	heads	TxByPriceAndTime
	signer  BaseSigner                          // Signer for the set of transactions
}

func NewTxsByPriceAndNonce(signer BaseSigner, txs map[account.Address][]*Transaction) *TxsByPriceAndNonce{
	heads := make(TxByPriceAndTime,0,len(txs))
	for from,accTxs := range txs {
		acc,_ := Sender(signer,accTxs[0])
		tx := accTxs[0]
		if acc != from {
			delete(txs,from)
		}
		heads = append(heads,tx)
		txs[from] = accTxs[1:]
	}
	heap.Init(&heads)
	return &TxsByPriceAndNonce{
		txs:		txs,
		heads:		heads,
		signer:		signer,
	}
}

// Peek returns the next transaction by price.
func (t *TxsByPriceAndNonce) Peek() *Transaction {
	if len(t.heads) == 0 {
		return nil
	}
	return t.heads[0]
}

// Shift replaces the current best head with the next one from the same account.
func (t *TxsByPriceAndNonce) Shift() {
	acc, _ := Sender(t.signer, t.heads[0])
	if txs, ok := t.txs[acc]; ok && len(txs) > 0 {
		wrapped := txs[0]
		t.heads[0], t.txs[acc] = wrapped, txs[1:]
		heap.Fix(&t.heads, 0)
		return
	}
	heap.Pop(&t.heads)
}
// Pop removes the best transaction, *not* replacing it with the next one from
// the same account. This should be used when a transaction cannot be executed
// and hence all subsequent ones should be discarded from the same account.
func (t *TxsByPriceAndNonce) Pop() {
	heap.Pop(&t.heads)
}
