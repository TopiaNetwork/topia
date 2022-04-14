package history

import (
	tpchaintypes "github.com/TopiaNetwork/topia/chain/types"
	"path/filepath"

	"github.com/TopiaNetwork/topia/ledger/backend"
	tplgtypes "github.com/TopiaNetwork/topia/ledger/types"
	tplog "github.com/TopiaNetwork/topia/log"
	tplogcmm "github.com/TopiaNetwork/topia/log/common"
)

type HistoryStore struct {
	log tplog.Logger
	backend.Backend
}

func NewHistoryStore(log tplog.Logger, rootPath string, backendType backend.BackendType) *HistoryStore {
	bsLog := tplog.CreateModuleLogger(tplogcmm.InfoLevel, "HistoryStore", log)
	backend := backend.NewBackend(backendType, bsLog, filepath.Join(rootPath, "historystore"), "historystore")

	return &HistoryStore{
		bsLog,
		backend,
	}
}

func (store *HistoryStore) Commit(block *tpchaintypes.Block) error {
	panic("implement me")
}

func (store *HistoryStore) GetHistoryForKey(namespace string, key string) (tplgtypes.ResultsIterator, error) {
	panic("implement me")
}
