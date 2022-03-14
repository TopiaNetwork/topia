package consensus

import (
	tpcmm "github.com/TopiaNetwork/topia/common"
	tptypes "github.com/TopiaNetwork/topia/common/types"
)

type consensusStore interface {
	ChainID() tpcmm.ChainID

	GetLatestBlock() (*tptypes.Block, error)

	SaveBlockMiddleResult(round uint64, blockResult *tptypes.BlockResultStoreInfo) error

	Commit(block *tptypes.Block) error

	ClearBlockMiddleResult(round uint64) error

	GetAllConsensusNodes() ([]string, error)

	GetActiveExecutorIDs() ([]string, error)

	GetActiveProposerIDs() ([]string, error)

	GetActiveValidatorIDs() ([]string, error)

	GetChainTotalWeight() (uint64, error)

	GetNodeWeight(nodeID string) (uint64, error)
}
