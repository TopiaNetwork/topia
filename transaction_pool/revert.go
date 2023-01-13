package transactionpool

import (
	"fmt"
	"time"

	"golang.org/x/sync/errgroup"

	tpchaintypes "github.com/TopiaNetwork/topia/chain/types"
	tpconfig "github.com/TopiaNetwork/topia/configuration"
	"github.com/TopiaNetwork/topia/network/message"
	txbasic "github.com/TopiaNetwork/topia/transaction/basic"
)

func (pool *transactionPool) validateTxsForRevert(tx *txbasic.Transaction) message.ValidationResult {
	if int64(tx.Size()) > tpconfig.DefaultTransactionPoolConfig().MaxSizeOfEachTx {
		pool.log.Errorf("transaction size is up to the TxMaxSize")
		return message.ValidationReject
	}
	if tx.Head.Nonce > MaxUint64 {
		pool.log.Errorf("transaction nonce is up to the MaxUint64")
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

func (pool *transactionPool) addTxsForBlocksRevert(blocks []*tpchaintypes.Block) {
	defer func(t0 time.Time) {
		pool.log.Infof("addTxsForBlocksRevert cost time: ", time.Since(t0))
	}(time.Now())

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
						var tx txbasic.Transaction
						if err := pool.marshaler.Unmarshal(txBytes, &tx); err != nil {
							err := fmt.Errorf("Unmarshal tx of block %d err: %v", block.Head.Height, err)
							pool.log.Errorf("%v", err)
							return err
						}

						if pool.validateTxsForRevert(&tx) == message.ValidationAccept {
							txList = append(txList, &tx)
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

	for _, tx := range txList {
		pool.addTx(tx, 0)
	}
}
