package ledger

import (
	"path/filepath"

	"github.com/TopiaNetwork/topia/ledger/backend"
	"github.com/TopiaNetwork/topia/ledger/block"
	"github.com/TopiaNetwork/topia/ledger/history"
	tplgss "github.com/TopiaNetwork/topia/ledger/state"
	tplog "github.com/TopiaNetwork/topia/log"
	tplogcmm "github.com/TopiaNetwork/topia/log/common"
)

type LedgerID string

type Ledger interface {
	CreateStateStore() (tplgss.StateStore, error)

	CreateStateStoreReadonly() (tplgss.StateStore, error)

	PendingStateStore() int32

	GetBlockStore() block.BlockStore

	IsGenesisState() bool
}

type ledger struct {
	id             LedgerID
	log            tplog.Logger
	backendStateDB backend.Backend
	blockStore     block.BlockStore
	historyStore   *history.HistoryStore
}

func NewLedger(chainDir string, id LedgerID, log tplog.Logger, backendType backend.BackendType) Ledger {
	rootPath := filepath.Join(chainDir, string(id))

	bsLog := tplog.CreateModuleLogger(tplogcmm.InfoLevel, "StateStore", log)
	backendStateDB := backend.NewBackend(backendType, bsLog, filepath.Join(rootPath, "statestore"), "statestore")

	return &ledger{
		id:             id,
		log:            log,
		backendStateDB: backendStateDB,
		blockStore:     block.NewBlockStore(log, rootPath, backendType),
		historyStore:   history.NewHistoryStore(log, rootPath, backendType),
	}
}

func (l *ledger) CreateStateStore() (tplgss.StateStore, error) {
	bsLog := tplog.CreateModuleLogger(tplogcmm.InfoLevel, "StateStore", l.log)
	return tplgss.NewStateStore(bsLog, l.backendStateDB, tplgss.Flag_ReadOnly|tplgss.Flag_WriteOnly), nil
}

func (l *ledger) CreateStateStoreReadonly() (tplgss.StateStore, error) {
	bsLog := tplog.CreateModuleLogger(tplogcmm.InfoLevel, "StateStore", l.log)
	return tplgss.NewStateStore(bsLog, l.backendStateDB, tplgss.Flag_ReadOnly), nil
}

func (l *ledger) PendingStateStore() int32 {
	return l.backendStateDB.PendingTxCount()
}

func (l *ledger) GetBlockStore() block.BlockStore {
	return l.blockStore
}

func (l *ledger) IsGenesisState() bool {
	reader := l.backendStateDB.Reader()
	defer reader.Discard()

	it, err := reader.Iterator(nil, nil)
	defer it.Close()
	if err != nil {
		l.log.Panicf("Backend state db reads iterator err: %v", err.Error())
	}
	if it.Next() {
		return true
	}

	return false
}
