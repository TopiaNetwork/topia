package ledger

import (
	"github.com/TopiaNetwork/topia/common/types"
	"path/filepath"

	"github.com/TopiaNetwork/topia/ledger/backend"
	"github.com/TopiaNetwork/topia/ledger/block"
	"github.com/TopiaNetwork/topia/ledger/history"
	"github.com/TopiaNetwork/topia/ledger/state"
	tplgtypes "github.com/TopiaNetwork/topia/ledger/types"
	tplog "github.com/TopiaNetwork/topia/log"
)

type LedgerID string

type Ledger interface {
	GetBlockByNumber(blockNum types.BlockNum) (*types.Block, error)

	GetBlocksIterator(startBlockNum types.BlockNum) (tplgtypes.ResultsIterator, error)

	TxIDExists(txID types.TxID) (bool, error)

	GetTransactionByID(txID types.TxID) (*types.Transaction, error)

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

func (l *ledger) GetBlockByNumber(blockNum types.BlockNum) (*types.Block, error) {
	//TODO implement me
	panic("implement me")
}

func (l *ledger) GetBlocksIterator(startBlockNum types.BlockNum) (tplgtypes.ResultsIterator, error) {
	//TODO implement me
	panic("implement me")
}

func (l *ledger) TxIDExists(txID types.TxID) (bool, error) {
	//TODO implement me
	panic("implement me")
}

func (l *ledger) GetTransactionByID(txID types.TxID) (*types.Transaction, error) {
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
