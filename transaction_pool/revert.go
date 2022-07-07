package transactionpool

import (
	"github.com/TopiaNetwork/topia/transaction"
	"sync/atomic"
	"time"

	tpchaintypes "github.com/TopiaNetwork/topia/chain/types"
	"github.com/TopiaNetwork/topia/network/message"
	txbasic "github.com/TopiaNetwork/topia/transaction/basic"
	txpooli "github.com/TopiaNetwork/topia/transaction_pool/interface"
)

func (pool *transactionPool) dropTxsForBlockAdded(newBlock *tpchaintypes.Block) {
	defer func(t0 time.Time) {
		pool.log.Infof("dropTxsForBlockAdded cost time: ", time.Since(t0))
	}(time.Now())

	for _, txByte := range newBlock.Data.Txs {
		var tx *txbasic.Transaction
		err := pool.marshaler.Unmarshal(txByte, &tx)
		if err != nil {
			pool.log.Errorf("Unmarshal tx error:", err)
		}
		txID, _ := tx.TxID()
		pool.RemoveTxByKey(txID, true)
	}
}

func (pool *transactionPool) addTxsForBlocksRevert(blocks []*tpchaintypes.Block) {
	defer func(t0 time.Time) {
		pool.log.Infof("addTxsForBlocksRevert cost time: ", time.Since(t0))
	}(time.Now())

	var addCnt int64
	var txList []*txbasic.Transaction
	for _, block := range blocks {
		for _, txByte := range block.Data.Txs {
			var tx *txbasic.Transaction
			err := pool.marshaler.Unmarshal(txByte, &tx)
			if err != nil {
				continue
			}
			if pool.validateTxsForRevert(tx) == message.ValidationAccept {
				addCnt += 1
				txList = append(txList, tx)
			}
		}
	}
	var removeCnt int64
	if addCnt <= (pool.config.TxPoolMaxCnt - pool.Count()) {
		removeCnt = 0
	} else {
		removeCnt = addCnt - (pool.config.TxPoolMaxCnt - pool.Count())
	}
	if removeCnt == int64(0) {
		pool.addTxs(txList, false)
	} else {
		isLocal := func(id txbasic.TxID) bool {
			tx, _ := pool.allWrappedTxs.Get(id)
			return tx.IsLocal
		}
		removeWrappedTx := func(id txbasic.TxID) {
			pool.allWrappedTxs.remoteTxs.Del(id)
		}
		removeCache := func(id txbasic.TxID) { pool.txCache.Remove(id) }
		remoteTxs := func() []*txbasic.Transaction {
			return pool.GetRemoteTxs()
		}
		cnt, isEnough := pool.pending.removeTxsForRevert(removeCnt, isLocal, removeWrappedTx, removeCache, remoteTxs)
		if isEnough {
			pool.addTxs(txList, false)
		} else {
			pool.log.Errorf("not enough space to revert blocks,need more :", cnt)
		}

	}

}
func (pool *transactionPool) validateTxsForRevert(tx *txbasic.Transaction) message.ValidationResult {

	if int64(tx.Size()) > txpooli.DefaultTransactionPoolConfig.MaxSizeOfEachTx {
		pool.log.Errorf("transaction size is up to the TxMaxSize")
		return message.ValidationReject
	}
	if tx.Head.Nonce > MaxUint64 {
		pool.log.Errorf("transaction nonce is up to the MaxUint64")
		return message.ValidationReject
	}
	if GasLimit(tx) < txpooli.DefaultTransactionPoolConfig.GasPriceLimit {
		pool.log.Errorf("transaction gasLimit is lower to GasPriceLimit")
		return message.ValidationReject
	}

	//*********Comment it out when testing******

	ac := transaction.CreatTransactionAction(tx)
	verifyResult := ac.Verify(pool.ctx, pool.log, pool.nodeId, nil)
	switch verifyResult {
	case txbasic.VerifyResult_Accept:
		return message.ValidationAccept
	case txbasic.VerifyResult_Ignore:
		return message.ValidationIgnore
	case txbasic.VerifyResult_Reject:
		return message.ValidationReject
	}
	//***************************
	return message.ValidationIgnore
}
func (pool *transactionPool) TruncateTxPool() {
	pool.pending = newPending()
	atomic.StoreInt64(&pool.poolSize, 0)
	atomic.StoreInt64(&pool.poolCount, 0)
	pool.allWrappedTxs = newAllLookupTxs()
	pool.txCache.Purge()
	pathIndex := pool.config.PathTxsStorage + "index.json"
	pathData := pool.config.PathTxsStorage + "data.json"
	pool.ClearLocalFile(pathIndex)
	pool.ClearLocalFile(pathData)
	pool.log.Tracef("TransactionPool Truncated")

}

//TxDifferenceList Delete set B from set A, return the differenct transaction list
func TxDifferenceList(a, b []*txbasic.Transaction) []*txbasic.Transaction {
	keep := make([]*txbasic.Transaction, 0, len(a))

	remove := make(map[txbasic.TxID]struct{})
	for _, tx := range b {
		txid, _ := tx.TxID()
		remove[txid] = struct{}{}
	}

	for _, tx := range a {
		txid, _ := tx.TxID()
		if _, ok := remove[txid]; !ok {
			keep = append(keep, tx)
		}
	}

	return keep
}
