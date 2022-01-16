package state

import (
	"github.com/TopiaNetwork/topia/ledger/backend"
	tplog "github.com/TopiaNetwork/topia/log"
	tplogcmm "github.com/TopiaNetwork/topia/log/common"
)

type StateStore struct {
	log    tplog.Logger
	dataS  stateData
	proofS stateProof
}

func NewStateStore(log tplog.Logger, rootPath string, backendType backend.BackendType) *StateStore {
	bsLog := tplog.CreateModuleLogger(tplogcmm.InfoLevel, "StateStore", log)
	//backend := backend.NewBackend(backendType, bsLog, filepath.Join(rootPath, "statestore"), "statestore")

	return &StateStore{
		log: bsLog,
	}
}

func (store *StateStore) Put(ns string, key string, value []byte) error {
	panic("implement me")
}

func (store *StateStore) Delete(ns string, key string) error {
	panic("implement me")
}

func (store *StateStore) Exists(ns string, key string) bool {
	panic("implement me")
}

func (store *StateStore) Update(ns string, key string) {
	panic("implement me")
}

func (store *StateStore) GetState(ns string, key string) ([]byte, error) {
	panic("implement me")
}
