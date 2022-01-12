package ledger

import (
	"path/filepath"

	"github.com/ethereum/go-ethereum/core/types"

	tpcmm "github.com/TopiaNetwork/topia/common"

	tptypes "github.com/TopiaNetwork/topia/common/types"
	"github.com/TopiaNetwork/topia/ledger/backend"
	"github.com/TopiaNetwork/topia/ledger/block"
	"github.com/TopiaNetwork/topia/ledger/history"
	"github.com/TopiaNetwork/topia/ledger/state"
	tplgtypes "github.com/TopiaNetwork/topia/ledger/types"
	tplog "github.com/TopiaNetwork/topia/log"
	"github.com/TopiaNetwork/topia/transaction"
)

type LedgerID string

type Ledger interface {
	ChainID() tpcmm.ChainID

	GetLatestBlock() (*tptypes.Block, error)

	SaveBlockMiddleResult(round uint64, blockResult *tptypes.BlockResultStoreInfo) error

	Commit() error

	ClearBlockMiddleResult(round uint64) error

	GetAllConsensusNodes() ([]string, error)

	GetBlockByNumber(blockNum tptypes.BlockNum) (*types.Block, error)

	GetBlocksIterator(startBlockNum tptypes.BlockNum) (tplgtypes.ResultsIterator, error)

	TxIDExists(txID transaction.TxID) (bool, error)

	GetTransactionByID(txID transaction.TxID) (*transaction.Transaction, error)

	GetBlockByHash(blockHash []byte) (*types.Block, error)

	GetBlockByTxID(txID string) (*types.Block, error)
}

type ledger struct {
	id           LedgerID
	log          tplog.Logger
	blockStore   *block.BlockStore
	historyStore *history.HistoryStore
	stateStore   *state.StateStore
}

func NewLedger(chainDir string, id LedgerID, log tplog.Logger, backendType backend.BackendType) Ledger {
	rootPath := filepath.Join(chainDir, string(id))

	return &ledger{
		id:           id,
		log:          log,
		blockStore:   block.NewBlockStore(log, rootPath, backendType),
		historyStore: history.NewHistoryStore(log, rootPath, backendType),
		stateStore:   state.NewStateStore(log, rootPath, backendType),
	}
}

func (l *ledger) ChainID() tpcmm.ChainID {
	//TODO implement me
	panic("implement me")
}

func (l *ledger) GetLatestBlock() (*tptypes.Block, error) {
	//TODO implement me
	panic("implement me")
}

func (l *ledger) SaveBlockMiddleResult(round uint64, blockResult *tptypes.BlockResultStoreInfo) error {
	//TODO implement me
	panic("implement me")
}

func (l *ledger) Commit() error {
	//TODO implement me
	panic("implement me")
}

func (l *ledger) ClearBlockMiddleResult(round uint64) error {
	//TODO implement me
	panic("implement me")
}

func (l *ledger) GetAllConsensusNodes() ([]string, error) {
	//TODO implement me
	panic("implement me")
}

func (l *ledger) GetBlockByNumber(blockNum tptypes.BlockNum) (*types.Block, error) {
	//TODO implement me
	panic("implement me")
}

func (l *ledger) GetBlocksIterator(startBlockNum tptypes.BlockNum) (tplgtypes.ResultsIterator, error) {
	//TODO implement me
	panic("implement me")
}

func (l *ledger) TxIDExists(txID transaction.TxID) (bool, error) {
	//TODO implement me
	panic("implement me")
}

func (l *ledger) GetTransactionByID(txID transaction.TxID) (*transaction.Transaction, error) {
	//TODO implement me
	panic("implement me")
}

func (l *ledger) GetBlockByHash(blockHash []byte) (*types.Block, error) {
	//TODO implement me
	panic("implement me")
}

func (l *ledger) GetBlockByTxID(txID string) (*types.Block, error) {
	//TODO implement me
	panic("implement me")
}
