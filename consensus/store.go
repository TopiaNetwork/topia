package consensus

import (
	tpcmm "github.com/TopiaNetwork/topia/common"
	tptypes "github.com/TopiaNetwork/topia/common/types"
	"github.com/TopiaNetwork/topia/transaction"
)

type consensusStore interface {
	ChainID() tpcmm.ChainID
	GetLatestBlock() (*tptypes.Block, error)
	SaveBlockMiddleResult(round uint64, blockResult *transaction.BlockResultStoreInfo) error
	Commit() error
	ClearBlockMiddleResult(round uint64) error
	GetAllConsensusNodes() ([]string, error)
}
