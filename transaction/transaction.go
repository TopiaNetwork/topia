package transaction

import (
	"container/heap"
	"encoding/hex"
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
