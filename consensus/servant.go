package consensus

import (
	tpcmm "github.com/TopiaNetwork/topia/chain"
	"github.com/TopiaNetwork/topia/chain/types"
)

type consensusServant interface {
	ChainID() tpcmm.ChainID

	GetLatestBlock() (*types.Block, error)

	GetAllConsensusNodes() ([]string, error)

	GetActiveExecutorIDs() ([]string, error)

	GetActiveProposerIDs() ([]string, error)

	GetActiveValidatorIDs() ([]string, error)

	GetChainTotalWeight() (uint64, error)

	GetNodeWeight(nodeID string) (uint64, error)
}
