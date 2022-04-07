package state

import (
	"bytes"
	"crypto/sha256"
	"encoding/gob"
	"errors"

	"github.com/lazyledger/smt"

	tplgcmm "github.com/TopiaNetwork/topia/ledger/backend/common"
)

func encodeProof(sp *smt.SparseMerkleProof) ([]byte, error) {
	var data bytes.Buffer
	enc := gob.NewEncoder(&data)
	err := enc.Encode(sp)

	return data.Bytes(), err
}

func decodeProof(proofData []byte) (*smt.SparseMerkleProof, error) {
	dec := gob.NewDecoder(bytes.NewBuffer(proofData))
	var proof smt.SparseMerkleProof
	err := dec.Decode(&proof)
	if err != nil {
		return nil, err
	}
	return &proof, nil
}

type stateProof struct {
	smTree *smt.SparseMerkleTree
}

func newStateProof(nodes tplgcmm.DBReadWriter, values tplgcmm.DBReadWriter) *stateProof {
	hasher := sha256.New()
	smTree := smt.NewSparseMerkleTree(&stateProofDB{nodes}, &stateProofDB{values}, hasher)
	stateIt, err := values.Iterator(nil, nil)
	defer stateIt.Close()
	if err != nil {
		return nil
	}
	for stateIt.Next() {
		smTree.Update(stateIt.Key(), stateIt.Value())
	}
	stateRoot := smTree.Root()
	if stateRoot != nil {
		smTree = smt.ImportSparseMerkleTree(&stateProofDB{nodes}, &stateProofDB{values}, hasher, stateRoot)
	}

	return &stateProof{
		smTree: smTree,
	}
}

func newStateProofReadonly(nodes tplgcmm.DBReader, values tplgcmm.DBReader) *stateProof {
	hasher := sha256.New()
	smTree := smt.NewSparseMerkleTree(&stateProofDBReadonly{nodes}, &stateProofDBReadonly{values}, hasher)
	stateIt, err := values.Iterator(nil, nil)
	if err != nil {
		return nil
	}
	for stateIt.Next() {
		smTree.Update(stateIt.Key(), stateIt.Value())
	}
	stateIt.Close()

	stateRoot := smTree.Root()
	if stateRoot != nil {
		smTree = smt.ImportSparseMerkleTree(&stateProofDBReadonly{nodes}, &stateProofDBReadonly{values}, hasher, stateRoot)
	}

	return &stateProof{
		smTree: smTree,
	}
}

func (p *stateProof) Get(key []byte) ([]byte, error) {
	return p.smTree.Get(key)
}

func (p *stateProof) SetWithNewRoot(key []byte, value []byte) ([]byte, error) {
	return p.smTree.Update(key, value)
}

func (p *stateProof) DeleteWithNewRoot(key []byte) ([]byte, error) {
	return p.smTree.Delete(key)
}

func (p *stateProof) Has(key []byte) (bool, error) {
	return p.smTree.Has(key)
}

func (p *stateProof) Root() []byte {
	return p.smTree.Root()
}

func (p *stateProof) Proof(key []byte) ([]byte, error) {
	proof, err := p.smTree.Prove(key)
	if err != nil {
		return nil, err
	}

	return encodeProof(&proof)
}

type stateProofDB struct {
	backendRW tplgcmm.DBReadWriter
}

type stateProofDBReadonly struct {
	backendR tplgcmm.DBReader
}

func (s *stateProofDB) Get(key []byte) ([]byte, error) {
	return s.backendRW.Get(key)
}

func (s *stateProofDB) Set(key []byte, value []byte) error {
	return s.backendRW.Set(key, value)
}

func (s *stateProofDB) Delete(key []byte) error {
	return s.backendRW.Delete(key)
}

func (s *stateProofDBReadonly) Get(key []byte) ([]byte, error) {
	return s.backendR.Get(key)
}

func (s *stateProofDBReadonly) Set(key []byte, value []byte) error {
	return errors.New("Can't set because of read only state proof db")
}

func (s *stateProofDBReadonly) Delete(key []byte) error {
	return errors.New("Can't delete because of read only state proof db")
}

func VerifyProof(proofData []byte, root []byte, key []byte, value []byte) bool {
	proof, err := decodeProof(proofData)
	if err != nil {
		return false
	}

	return smt.VerifyProof(*proof, root, key, value, sha256.New())
}
