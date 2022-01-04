package state

import (
	"path/filepath"

	"github.com/TopiaNetwork/topia/common/types"
	"github.com/TopiaNetwork/topia/ledger/backend"
	tplog "github.com/TopiaNetwork/topia/log"
	tplogcmm "github.com/TopiaNetwork/topia/log/common"
	"github.com/TopiaNetwork/topia/transaction"
)

type TxState struct {
	BlockNum types.BlockNum
	TxID     transaction.TxID
}

type VersionedValue struct {
	Value    []byte
	Metadata []byte
	Version  *TxState
}

type StateStore struct {
	log     tplog.Logger
	backend backend.Backend
}

func NewStateStore(log tplog.Logger, rootPath string, backendType backend.BackendType) *StateStore {
	bsLog := tplog.CreateModuleLogger(tplogcmm.InfoLevel, "StateStore", log)
	backend := backend.NewBackend(backendType, bsLog, filepath.Join(rootPath, "statestore"), "statestore")

	return &StateStore{
		bsLog,
		backend,
	}
}

func (store *StateStore) Put(ns string, key string, value []byte, metadata []byte, version *TxState) error {
	panic("implement me")
}

func (store *StateStore) Delete(ns string, key string, version *TxState) error {
	panic("implement me")
}

func (store *StateStore) Exists(ns string, key string) bool {
	panic("implement me")
}

func (store *StateStore) Update(ns string, key string, vv *VersionedValue) {
	panic("implement me")
}

func (store *StateStore) GetState(ns string, key string) *VersionedValue {
	panic("implement me")
}

func (store *StateStore) GetVersion(namespace string, key string) (*TxState, error) {
	panic("implement me")
}
