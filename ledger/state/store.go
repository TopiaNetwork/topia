package state

import (
	"errors"
	tplgcmm "github.com/TopiaNetwork/topia/ledger/backend/common"
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
	log       tplog.Logger
	backendR  tplgcmm.DBReader
	backendRW tplgcmm.DBReadWriter
	dataS     *stateData
	proofS    *stateProof
}

func newStateStoreComposition(log tplog.Logger, backendRW tplgcmm.DBReadWriter, name string, cacheSize int) *StateStoreComposition {
	backendDBNamed := backend.NewBackendRWPrefixed([]byte(name), backendRW)

	stateDataBackend := backend.NewBackendRWPrefixed([]byte{stateDataPrefix}, backendDBNamed)
	stateData := newStateData(name, stateDataBackend, cacheSize)

	smtNodeDataBackend := backend.NewBackendRWPrefixed([]byte{merkleNodePrefix}, backendDBNamed)
	smtNodeValueBackend := backend.NewBackendRWPrefixed([]byte{merkleValuePrefix}, backendDBNamed)
	stateProof := newStateProof(smtNodeDataBackend, smtNodeValueBackend)

	return &StateStoreComposition{
		log:       log,
		backendRW: backendRW,
		dataS:     stateData,
		proofS:    stateProof,
	}
}

func newStateStoreCompositionReadOnly(log tplog.Logger, backendR tplgcmm.DBReader, name string, cacheSize int) *StateStoreComposition {
	backendDBNamed := backend.NewBackendRPrefixed([]byte(name), backendR)

	stateDataBackend := backend.NewBackendRPrefixed([]byte{stateDataPrefix}, backendDBNamed)
	stateData := newStateDataReadonly(name, stateDataBackend, cacheSize)

	smtNodeDataBackend := backend.NewBackendRPrefixed([]byte{merkleNodePrefix}, backendDBNamed)
	smtNodeValueBackend := backend.NewBackendRPrefixed([]byte{merkleValuePrefix}, backendDBNamed)
	stateProof := newStateProofReadonly(smtNodeDataBackend, smtNodeValueBackend)

	return &StateStoreComposition{
		log:      log,
		backendR: backendR,
		dataS:    stateData,
		proofS:   stateProof,
	}
}

func (store *StateStoreComposition) Root() []byte {
	return store.proofS.Root()
}

func (store *StateStoreComposition) Put(key []byte, value []byte) error {
	if store.backendR != nil {
		return errors.New("Can't put because of read only state store composition")
	}

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
	if store.backendR != nil {
		return errors.New("Can't delete because of read only state store composition")
	}

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
	return store.dataS.Has(key)
}

func (store *StateStoreComposition) Update(key []byte, value []byte) error {
	if store.backendR != nil {
		return errors.New("Can't update because of read only state store composition")
	}

	return store.Put(key, value)
}

func (store *StateStoreComposition) GetStateData(key []byte) ([]byte, error) {
	return store.dataS.Get(key)
}

func (store *StateStoreComposition) GetState(key []byte) ([]byte, []byte, error) {
	var rError error

	value, err1 := store.dataS.Get(key)
	if err1 != nil {
		rError = multierror.Append(rError, err1)
	}

	proof, err2 := store.proofS.Proof(key)
	if err2 != nil {
		rError = multierror.Append(rError, err2)
	}

	return value, proof, rError
}

func (store *StateStoreComposition) GetAllStateData() ([][]byte, [][]byte, error) {
	var keys [][]byte
	var values [][]byte

	dataIt, err := store.dataS.Iterator(nil, nil)
	if err != nil {
		return nil, nil, err
	}
	defer dataIt.Close()

	for dataIt.Next() {
		keys = append(keys, dataIt.Key())
		values = append(values, dataIt.Value())
	}

	return keys, values, err
}

func (store *StateStoreComposition) IterateAllStateDataCB(iterCBFunc IterStateDataCBFunc) error {
	dataIt, err := store.dataS.Iterator(nil, nil)
	if err != nil {
		return err
	}
	defer dataIt.Close()

	for dataIt.Next() {
		iterCBFunc(dataIt.Key(), dataIt.Value())
	}

	return nil
}

func (store *StateStoreComposition) GetAllState() ([][]byte, [][]byte, [][]byte, error) {
	var keys [][]byte
	var values [][]byte
	var proofs [][]byte

	dataIt, err := store.dataS.Iterator(nil, nil)
	if err != nil {
		return nil, nil, nil, err
	}
	defer dataIt.Close()

	for dataIt.Next() {
		keys = append(keys, dataIt.Key())
		values = append(values, dataIt.Value())

		proof, _, err := store.GetState(dataIt.Key())
		if err != nil {
			return nil, nil, nil, err
		}
		proofs = append(proofs, proof)
	}

	return keys, values, proofs, err
}

func (store *StateStoreComposition) IterateAllStateCB(iterCBFunc IterStateCBFunc) error {
	dataIt, err := store.dataS.Iterator(nil, nil)
	if err != nil {
		return err
	}
	defer dataIt.Close()

	for dataIt.Next() {
		proof, _, err := store.GetState(dataIt.Key())
		if err != nil {
			return err
		}

		iterCBFunc(dataIt.Key(), dataIt.Value(), proof)
	}

	return nil
}

func (store *StateStoreComposition) Commit() error {
	if store.backendR != nil {
		return errors.New("Can't commit because of read only state store composition")
	}

	return store.backendRW.Commit()
}
