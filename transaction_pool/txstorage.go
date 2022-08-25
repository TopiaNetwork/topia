package transactionpool

import (
	"os"
	"sync"

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
	var allTxInfo []*wrappedTxData
	isLocalsNil := func() bool {
		return pool.allWrappedTxs.localTxs.Size() == 0
	}
	saveAllLocals := func() error {
		getAllTxInfo := func(k interface{}, v interface{}) {
			TxInfo := v.(*wrappedTx)
			txByte, _ := pool.marshaler.Marshal(TxInfo.Tx)
			txData := &wrappedTxData{
				TxID:          TxInfo.TxID,
				IsLocal:       TxInfo.IsLocal,
				LastTime:      TxInfo.LastTime,
				LastHeight:    TxInfo.LastHeight,
				TxState:       TxInfo.TxState,
				IsRepublished: TxInfo.IsRepublished,
				Tx:            txByte,
			}
			allTxInfo = append(allTxInfo, txData)
		}

		pool.allWrappedTxs.localTxs.IterateCallback(getAllTxInfo)

		pathIndex := path + "index.json"
		pathData := path + "data.json"
		fileIndex, err := os.OpenFile(pathIndex, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, 0666)
		if err != nil {
			return err
		}
		fileData, err := os.OpenFile(pathData, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, 0666)

		if err != nil {
			return err
		}
		txIndex := newTxStorageIndex()
		for _, v := range allTxInfo {
			ByteV, _ := pool.marshaler.Marshal(v)
			n, _ := fileData.Write(ByteV)
			txIndex.set(string(v.TxID), [2]int{txIndex.LastPos, n})
		}
		txInByte, _ := pool.marshaler.Marshal(txIndex)
		fileIndex.Write(txInByte)
		fileIndex.Close()
		fileData.Close()
		return nil
	}
	return pool.txServant.saveAllLocalTxs(isLocalsNil, saveAllLocals)
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
