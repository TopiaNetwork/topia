package chain

import (
	tpchainar "github.com/TopiaNetwork/topia/chain/archiver"
	tpchaintypes "github.com/TopiaNetwork/topia/chain/types"
	tpchainuni "github.com/TopiaNetwork/topia/chain/universal"
	tpcmm "github.com/TopiaNetwork/topia/common"
	tplog "github.com/TopiaNetwork/topia/log"
)

type BlockCollector interface {
	Collect(block *tpchaintypes.Block, blockRS *tpchaintypes.BlockResult) (*tpchaintypes.Block, *tpchaintypes.BlockResult, error)
}

func CreateBlockCollector(log tplog.Logger, nodeID string, nodeRole tpcmm.NodeRole) BlockCollector {
	if nodeRole&tpcmm.NodeRole_Archiver == tpcmm.NodeRole_Archiver {
		return tpchainar.NewBlockCollector(log, nodeID)
	} else {
		return tpchainuni.NewBlockCollector(log, nodeID)
	}
}
