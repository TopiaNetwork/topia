package state

import (
	"github.com/hashicorp/go-multierror"

	"github.com/TopiaNetwork/topia/ledger/backend"
	tplog "github.com/TopiaNetwork/topia/log"
)

const (
	MOD_NAME = "StateStore"
)

const (
	stateMerkleRootKey = byte(0x00) // Key for root hashes of Merkle trees
	stateDataPrefix    = byte(0x01) // Prefix for state data
	indexPrefix        = byte(0x02) // Prefix for Store reverse index
	merkleNodePrefix   = byte(0x03) // Prefix for Merkle tree nodes
	merkleValuePrefix  = byte(0x04) // Prefix for Merkle tree values
)

type StateStoreComposition struct {
	log    tplog.Logger
	dataS  *stateData
	proofS *stateProof
}

func newStateStoreComposition(log tplog.Logger, backendDB backend.Backend, name string) *StateStoreComposition {
	backendDBNamed := backend.NewBackendPrefixed([]byte(name), backendDB)

	stateDataBackend := backend.NewBackendPrefixed([]byte{stateDataPrefix}, backendDBNamed)
	stateData := newStateData(name, stateDataBackend)

	smtNodeDataBackend := backend.NewBackendPrefixed([]byte{merkleNodePrefix}, backendDBNamed)
	smtNodeValueBackend := backend.NewBackendPrefixed([]byte{merkleValuePrefix}, backendDBNamed)
	stateProof := newStateProof(smtNodeDataBackend, smtNodeValueBackend)

	return &StateStoreComposition{
		log:    log,
		dataS:  stateData,
		proofS: stateProof,
	}
}

func (store *StateStoreComposition) Put(key []byte, value []byte) error {
	var rError error

	if err1 := store.dataS.Set(key, value); err1 != nil {
		rError = multierror.Append(rError, err1)
	}

	if _, err2 := store.proofS.SetWithNewRoot(key, value); err2 != nil {
		rError = multierror.Append(rError, err2)
	}

	return rError
}

func (store *StateStoreComposition) Delete(key []byte) error {
	var rError error

	if err1 := store.dataS.Delete(key); err1 != nil {
		rError = multierror.Append(rError, err1)
	}

	if _, err2 := store.proofS.DeleteWithNewRoot(key); err2 != nil {
		rError = multierror.Append(rError, err2)
	}

	return rError
}

func (store *StateStoreComposition) Exists(key []byte) (bool, error) {
	return store.dataS.Has(key, nil)
}

func (store *StateStoreComposition) Update(key []byte, value []byte) error {
	return store.Put(key, value)
}

func (store *StateStoreComposition) GetState(key []byte) ([]byte, []byte, error) {
	var rError error

	value, err1 := store.dataS.Get(key, nil)
	if err1 != nil {
		rError = multierror.Append(rError, err1)
	}

	proof, err2 := store.proofS.Proof(key)
	if err2 != nil {
		rError = multierror.Append(rError, err2)
	}

	return value, proof, rError
}
