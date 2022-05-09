package service

import (
	tpchaintypes "github.com/TopiaNetwork/topia/chain/types"
	tplgblock "github.com/TopiaNetwork/topia/ledger/block"
	txbasic "github.com/TopiaNetwork/topia/transaction/basic"
)

type BlockService interface {
	GetTransactionByID(txID txbasic.TxID) (*txbasic.Transaction, error)

	GetTransactionResultByID(txID txbasic.TxID) (*txbasic.TransactionResult, error)

	GetBlockByHash(blockHash tpchaintypes.BlockHash) (*tpchaintypes.Block, error)

	GetBlockByTxID(txID txbasic.TxID) (*tpchaintypes.Block, error)

	GetBlockByNumber(blockNum tpchaintypes.BlockNum) (*tpchaintypes.Block, error)

	GetBatchBlocks(startBlockNum tpchaintypes.BlockNum, count uint64) ([]*tpchaintypes.Block, error)
}

type blockService struct {
	tplgblock.BlockStore
}

func (bs *blockService) GetBatchBlocks(startBlockNum tpchaintypes.BlockNum, count uint64) ([]*tpchaintypes.Block, error) {
	qIt, err := bs.GetBlocksIterator(startBlockNum)
	if err != nil {
		return nil, err
	}
	defer qIt.Close()

	var returnBlock []*tpchaintypes.Block
	for count > 0 {
		nextBlockI, err := qIt.Next()
		if err != nil {
			return nil, err
		}
		if nextBlockI == nil {
			break
		}

		if nextBlock, ok := nextBlockI.(*tpchaintypes.Block); ok {
			returnBlock = append(returnBlock, nextBlock)
			count--
		}
	}

	return returnBlock, nil
}
