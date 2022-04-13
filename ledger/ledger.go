package ledger

import (
	"path/filepath"

	"go.uber.org/atomic"

	tpcmm "github.com/TopiaNetwork/topia/common"
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

	UpdateState(state tpcmm.LedgerState)

	PendingStateStore() int32

	GetBlockStore() block.BlockStore

	State() tpcmm.LedgerState
}

type ledger struct {
	id             LedgerID
	log            tplog.Logger
	state          atomic.Value
	backendStateDB backend.Backend
	blockStore     block.BlockStore
	historyStore   *history.HistoryStore
}

func NewLedger(chainDir string, id LedgerID, log tplog.Logger, backendType backend.BackendType) Ledger {
	rootPath := filepath.Join(chainDir, string(id))

	bsLog := tplog.CreateModuleLogger(tplogcmm.InfoLevel, "StateStore", log)
	backendStateDB := backend.NewBackend(backendType, bsLog, filepath.Join(rootPath, "statestore"), "statestore")

	l := &ledger{
		id:             id,
		log:            log,
		backendStateDB: backendStateDB,
		blockStore:     block.NewBlockStore(log, rootPath, backendType),
		historyStore:   history.NewHistoryStore(log, rootPath, backendType),
	}

	l.state.Store(tpcmm.LedgerState_Uninitialized)

	return l
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

func (l *ledger) State() tpcmm.LedgerState {
	return l.state.Load().(tpcmm.LedgerState)
}

func (l *ledger) UpdateState(state tpcmm.LedgerState) {
	l.state.Swap(state)
}
