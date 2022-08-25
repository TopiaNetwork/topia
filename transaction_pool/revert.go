package transactionpool

import (
	"fmt"
	"golang.org/x/sync/errgroup"
	"sync/atomic"
	"time"

	tpchaintypes "github.com/TopiaNetwork/topia/chain/types"
	tpcrtypes "github.com/TopiaNetwork/topia/crypt/types"
	"github.com/TopiaNetwork/topia/network/message"
	txbasic "github.com/TopiaNetwork/topia/transaction/basic"
	txpooli "github.com/TopiaNetwork/topia/transaction_pool/interface"
)

func (pool *transactionPool) dropTxsForBlockAdded(newBlock *tpchaintypes.Block) {
	defer func(t0 time.Time) {
		pool.log.Infof("dropTxsForBlockAdded cost time: ", time.Since(t0))
	}(time.Now())

	var egChunk errgroup.Group
	for _, dataChunkBytes := range newBlock.Data.DataChunks {
		dataChunkBytes := dataChunkBytes

		egChunk.Go(func() error {
			var dataChunk tpchaintypes.BlockDataChunk
			if err := dataChunk.Unmarshal(dataChunkBytes); err != nil {
				pool.log.Errorf("Unmarshal data chunk of block %d err: %v", newBlock.Head.Height, err)
			}

			var txIDs []txbasic.TxID
			for _, txByte := range dataChunk.Txs {
				var tx *txbasic.Transaction
				err := pool.marshaler.Unmarshal(txByte, &tx)
				if err != nil {
					pool.log.Errorf("Unmarshal tx error:", err)
				}
				txID, _ := tx.TxID()
				txIDs = append(txIDs, txID)
			}
			pool.RemoveTxBatch(txIDs)

			return nil
		})
		egChunk.Wait()
	}
}

func (pool *transactionPool) addTxsForBlocksRevert(blocks []*tpchaintypes.Block) {
	defer func(t0 time.Time) {
		pool.log.Infof("addTxsForBlocksRevert cost time: ", time.Since(t0))
	}(time.Now())

	var addCnt int64
	var txList []*txbasic.Transaction
	var eg errgroup.Group
	for _, block := range blocks {
		block := block
		eg.Go(func() error {
			var egChunk errgroup.Group
			for _, dataChunkBytes := range block.Data.DataChunks {
				dataChunkBytes := dataChunkBytes
				egChunk.Go(func() error {
					var dataChunk tpchaintypes.BlockDataChunk
					if err := dataChunk.Unmarshal(dataChunkBytes); err != nil {
						err := fmt.Errorf("Unmarshal data chunk of block %d err: %v", block.Head.Height, err)
						pool.log.Errorf("%v", err)
						return err
					}

					for _, txBytes := range dataChunk.Txs {
						var tx *txbasic.Transaction
						if err := pool.marshaler.Unmarshal(txBytes, &tx); err != nil {
							err := fmt.Errorf("Unmarshal tx of block %d err: %v", block.Head.Height, err)
							pool.log.Errorf("%v", err)
							return err
						}

						if pool.validateTxsForRevert(tx) == message.ValidationAccept {
							addCnt += 1
							txList = append(txList, tx)
						}
					}

					return nil
				})
			}
			return egChunk.Wait()
		})
	}
	err := eg.Wait()
	if err != nil {
		return
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

		rmTxInfo := func(id txbasic.TxID) {
			pool.allWrappedTxs.remoteTxs.Del(id)
			pool.txCache.Remove(id)
			pool.chanDelTxsStorage <- []txbasic.TxID{id}
		}
		delSorted := func(addr tpcrtypes.Address) {

			for {
				old := atomic.LoadInt32(&pool.isPicking)
				if atomic.CompareAndSwapInt32(&pool.isPicking, old, int32(1)) {
					break
				}
			}
			pool.sortedTxs.removeAddr(addr)

			for {
				old := atomic.LoadInt32(&pool.isPicking)
				if atomic.CompareAndSwapInt32(&pool.isPicking, old, int32(0)) {
					break
				}
			}
		}
		addSorted := func(addr tpcrtypes.Address, maxPrice uint64, isMaxChanged bool, txs []*txbasic.Transaction) bool {

			if atomic.LoadInt32(&pool.isPicking) == int32(1) {
				sortItem := &sortedItem{
					account:           addr,
					maxPrice:          maxPrice,
					isMaxPriceChanged: isMaxChanged,
					txs:               txs,
				}
				pool.chanSortedItem <- sortItem
				return false
			}

			pool.sortedTxs.addAccTx(addr, maxPrice, isMaxChanged, txs)
			return true
		}
		remoteTxs := func() []*txbasic.Transaction {
			return pool.GetRemoteTxs()
		}
		for {
			old := atomic.LoadInt32(&pool.isInRemove)
			if atomic.CompareAndSwapInt32(&pool.isInRemove, old, 1) {
				break
			}
		}
		for {
			old := atomic.LoadInt32(&pool.isPicking)
			if atomic.CompareAndSwapInt32(&pool.isPicking, old, 1) {
				break
			}
		}

		cnt, isEnough := pool.pending.removeTxsForRevert(removeCnt, isLocal, rmTxInfo, delSorted, addSorted, remoteTxs)
		for {
			old := atomic.LoadInt32(&pool.isInRemove)
			if atomic.CompareAndSwapInt32(&pool.isInRemove, old, 0) {
				break
			}
		}
		for {
			old := atomic.LoadInt32(&pool.isPicking)
			if atomic.CompareAndSwapInt32(&pool.isPicking, old, 0) {
				break
			}
		}

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
	//
	//ac := transaction.CreatTransactionAction(tx)
	//verifyResult := ac.Verify(pool.ctx, pool.log, pool.nodeId, nil)
	//switch verifyResult {
	//case txbasic.VerifyResult_Accept:
	//	return message.ValidationAccept
	//case txbasic.VerifyResult_Ignore:
	//	return message.ValidationIgnore
	//case txbasic.VerifyResult_Reject:
	//	return message.ValidationReject
	//}
	//***************************
	return message.ValidationAccept
}
func (pool *transactionPool) TruncateTxPool() {
	pool.pending = newAccTxs()
	pool.prepareTxs = newAccTxs()
	pool.allWrappedTxs = newAllLookupTxs()
	pool.pendingNonces = newAccountNonce(pool.txServant.getStateQuery())
	atomic.StoreInt64(&pool.pendingCount, 0)
	atomic.StoreInt64(&pool.pendingSize, 0)
	atomic.StoreInt64(&pool.poolCount, 0)
	atomic.StoreInt64(&pool.poolSize, 0)
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
