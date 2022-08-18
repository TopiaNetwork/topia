package universal

import (
	tpchaintypes "github.com/TopiaNetwork/topia/chain/types"
	tplog "github.com/TopiaNetwork/topia/log"
)

type blockCollector struct {
}

func NewBlockCollector(log tplog.Logger, nodeID string) *blockCollector {
	return &blockCollector{}
}

func (bc *blockCollector) Collect(block *tpchaintypes.Block, blockRS *tpchaintypes.BlockResult) (*tpchaintypes.Block, *tpchaintypes.BlockResult, error) {
	return block, blockRS, nil
}
