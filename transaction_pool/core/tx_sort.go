package core

import (
	"sort"
	"sync"

	txbasic "github.com/TopiaNetwork/topia/transaction/basic"
)

type SortWay byte

const (
	SortWay_Unknown SortWay = iota
	SortWay_Misc
	SortWay_Height
	SortWay_Time
)

type sortedTxList struct {
	sync   sync.RWMutex
	sWay   SortWay
	txList []TxWrapper
}

func newSortedTxList(sWay SortWay) *sortedTxList {
	return &sortedTxList{
		sWay: sWay,
	}
}

func (st *sortedTxList) addTx(wTx TxWrapper) {
	st.sync.Lock()
	defer st.sync.Unlock()

	index := sort.Search(len(st.txList), func(i int) bool {
		return st.txList[i].Less(st.sWay, wTx)
	})

	if index == len(st.txList) {
		st.txList = append(st.txList, wTx)
		return
	}

	st.txList = append(st.txList[:index+1], st.txList[index:]...)
	st.txList[index] = wTx
}

func (st *sortedTxList) removeTx(txID txbasic.TxID) TxWrapper {
	st.sync.Lock()
	defer st.sync.Unlock()

	for i, wTx := range st.txList {
		if wTx.TxID() == txID {
			st.txList = append(st.txList[:i], st.txList[i+1:]...)
			return wTx
		}
	}

	return nil
}

func (st *sortedTxList) reset() {
	st.sync.Lock()
	defer st.sync.Unlock()

	st.txList = make([]TxWrapper, 0)
}

func (st *sortedTxList) size() int {
	st.sync.RLock()
	defer st.sync.RUnlock()

	return len(st.txList)
}

func (st *sortedTxList) getTx(txID txbasic.TxID) TxWrapper {
	st.sync.RLock()
	defer st.sync.RUnlock()

	for _, wTx := range st.txList {
		if wTx.TxID() == txID {
			return wTx
		}
	}

	return nil
}

func (st *sortedTxList) pickTxs(blockMaxBytes int64, blockMaxGas int64) []*txbasic.Transaction {
	st.sync.RLock()
	defer st.sync.RUnlock()

	if len(st.txList) == 0 {
		return nil
	}

	var totalSize int
	var rtnWTx []*txbasic.Transaction
	for _, wTx := range st.txList {
		totalSize += wTx.Size()
		if totalSize > int(blockMaxBytes) {
			break
		} else {
			rtnWTx = append(rtnWTx, wTx.OriginTx())

		}
	}

	return rtnWTx
}
