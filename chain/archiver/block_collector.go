package archiver

import (
	"bytes"
	"fmt"
	"sync"

	tpchaintypes "github.com/TopiaNetwork/topia/chain/types"
	tplog "github.com/TopiaNetwork/topia/log"
)

type collectedBlockInfo struct {
	block   *tpchaintypes.Block
	blockRS *tpchaintypes.BlockResult
}

type blockCollector struct {
	log       tplog.Logger
	nodeID    string
	sync      sync.RWMutex
	blockList []*collectedBlockInfo
}

func NewBlockCollector(log tplog.Logger, nodeID string) *blockCollector {
	return &blockCollector{
		log:    log,
		nodeID: nodeID,
	}
}

func (bc *blockCollector) isExistSameBlockData(origin *tpchaintypes.BlockData) bool {
	for _, target := range bc.blockList {
		if origin.Version != target.block.Data.Version {
			continue
		}

		for _, orginDataChunkBytes := range origin.DataChunks {
			for _, targetDataChunkBytes := range target.block.Data.DataChunks {
				if bytes.Equal(orginDataChunkBytes, targetDataChunkBytes) {
					return true
				}
			}
		}
	}

	return false
}

func (bc *blockCollector) Collect(block *tpchaintypes.Block, blockRS *tpchaintypes.BlockResult) (*tpchaintypes.Block, *tpchaintypes.BlockResult, error) {
	bc.sync.Lock()
	defer bc.sync.Unlock()

	bCount := len(bc.blockList)

	if bCount == 0 {
		bc.blockList = append(bc.blockList, &collectedBlockInfo{
			block:   block,
			blockRS: blockRS,
		})

		return nil, nil, nil
	}
	if block.Head.Height == bc.blockList[0].block.Head.Height {
		recvBlockBytes, _ := block.Head.Marshal()
		bhBytes, _ := bc.blockList[0].block.Head.Marshal()
		if bytes.Equal(recvBlockBytes, bhBytes) {
			if bc.isExistSameBlockData(block.Data) {
				err := fmt.Errorf("Received invalid block and block result: height %d, reason same data, self node %s", block.Head.Height, bc.nodeID)
				bc.log.Errorf("%v", err)
				return nil, nil, err
			}

			bc.blockList = append(bc.blockList, &collectedBlockInfo{
				block:   block,
				blockRS: blockRS,
			})
		} else {
			err := fmt.Errorf("Received invalid block and block result: height %d, reason different head, self node %s", block.Head.Height, bc.nodeID)
			bc.log.Errorf("%v", err)
			return nil, nil, err
		}
	} else if block.Head.Height == bc.blockList[0].block.Head.Height+1 {
		for i := 1; i < bCount; i++ {
			bc.blockList[0].block.Data.DataChunks = append(bc.blockList[0].block.Data.DataChunks, bc.blockList[i].block.Data.DataChunks...)
			bc.blockList[0].blockRS.Data.ResultDataChunks = append(bc.blockList[0].blockRS.Data.ResultDataChunks, bc.blockList[i].blockRS.Data.ResultDataChunks...)
		}

		rBlock := bc.blockList[0].block
		rBlockRS := bc.blockList[0].blockRS

		bc.blockList = bc.blockList[:0]
		bc.blockList = append(bc.blockList, &collectedBlockInfo{
			block:   block,
			blockRS: blockRS,
		})

		return rBlock, rBlockRS, nil
	} else {
		err := fmt.Errorf("Received invalid block and block result: received height %d, current height %d, self node %s", block.Head.Height, bc.blockList[0].block.Head.Height, bc.nodeID)
		bc.log.Errorf("%v", err)
		return nil, nil, err
	}

	return nil, nil, nil
}
