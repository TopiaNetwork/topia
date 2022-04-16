package consensus

import (
	tpchaintypes "github.com/TopiaNetwork/topia/chain/types"
	"github.com/TopiaNetwork/topia/common"
)

type consensusServant interface {
	ChainID() tpchaintypes.ChainID

	GetLatestEpoch() (*common.EpochInfo, error)

	GetLatestBlock() (*tpchaintypes.Block, error)

	GetAllConsensusNodeIDs() ([]string, error)

	GetActiveExecutorIDs() ([]string, error)

	GetActiveProposerIDs() ([]string, error)

	GetActiveValidatorIDs() ([]string, error)

	GetTotalWeight() (uint64, error)

	GetActiveExecutorsTotalWeight() (uint64, error)

	GetActiveProposersTotalWeight() (uint64, error)

	GetActiveValidatorsTotalWeight() (uint64, error)

	GetNodeWeight(nodeID string) (uint64, error)
}
