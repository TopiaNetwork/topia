package transactionpool

import (
	"os"
	"sync"

	tpcmm "github.com/TopiaNetwork/topia/common"
	txbasic "github.com/TopiaNetwork/topia/transaction/basic"
	txpooli "github.com/TopiaNetwork/topia/transaction_pool/interface"
)

func (pool *transactionPool) savePoolConfig(path string) error {
	return pool.txServant.savePoolConfig(path, pool.config, pool.marshaler)
}

func (pool *transactionPool) loadAndSetPoolConfig(path string) error {
	setConf := func(config txpooli.TransactionPoolConfig) {
		pool.config = config
	}

	return pool.txServant.loadAndSetPoolConfig(path, pool.marshaler, setConf)
}

func (pool *transactionPool) SaveAllLocalTxsData(path string) error {
	getLocals := func() *tpcmm.ShrinkableMap {
		return pool.allWrappedTxs.localTxs
	}
	return pool.txServant.saveAllLocalTxs(path, pool.marshaler, getLocals)
}
func (pool *transactionPool) SaveLocalTxsData(path string, wrappedTxs []*wrappedTx) error {

	return pool.txServant.saveLocalTxs(path, pool.marshaler, wrappedTxs)
}

func (pool *transactionPool) LoadLocalTxsData(path string) error {

	addLocalTx := func(txID txbasic.TxID, txWrap *wrappedTx) {
		pool.AddTx(txWrap.Tx, true)
		pool.allWrappedTxs.Set(txID, txWrap)
		pool.txCache.Add(txID, txWrap.TxState)
	}
	return pool.txServant.loadAndAddLocalTxs(path, pool.marshaler, addLocalTx)
}

func (pool *transactionPool) DelLocalTxsData(path string, ids []txbasic.TxID) error {
	return pool.txServant.delLocalTxs(path, pool.marshaler, ids)
}

func (pool *transactionPool) ClearLocalFile(path string) {
	if path != "" {
		if err := os.Remove(path); err != nil {
			pool.log.Errorf("Failed to remove local file", "err", err)
		} else {
			pool.log.Infof("loadTxsData file removed")
		}
	} else {
		pool.log.Errorf("Failed to load transactions data: config.PathTxsStorage is nil")
	}
}

type txStorageIndex struct {
	mu            sync.RWMutex
	TxBeginAndLen map[string][2]int
	LastPos       int
}

func newTxStorageIndex() *txStorageIndex {
	txIndex := &txStorageIndex{TxBeginAndLen: make(map[string][2]int, 0)}
	return txIndex
}
func (txIn *txStorageIndex) get(k string) ([2]int, bool) {
	txIn.mu.RLock()
	defer txIn.mu.RUnlock()
	if v, ok := txIn.TxBeginAndLen[k]; !ok {
		return [2]int{0, 0}, false
	} else {
		return v, true
	}
}

func (txIn *txStorageIndex) set(k string, v [2]int) {
	txIn.mu.Lock()
	defer txIn.mu.Unlock()
	txIn.TxBeginAndLen[k] = v
	txIn.LastPos += v[1]
}

func (txIn *txStorageIndex) del(k string) {
	txIn.mu.RLock()
	defer txIn.mu.RUnlock()
	delete(txIn.TxBeginAndLen, k)
}
