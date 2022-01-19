package state

import (
	"bytes"
	"crypto/sha256"
	"encoding/gob"

	"github.com/lazyledger/smt"

	"github.com/TopiaNetwork/topia/ledger/backend"
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

func newStateProof(nodes backend.Backend, values backend.Backend) *stateProof {
	return &stateProof{
		smTree: smt.NewSparseMerkleTree(&stateProofDB{nodes}, &stateProofDB{values}, sha256.New()),
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
	return p.Root()
}

func (p *stateProof) Proof(key []byte) ([]byte, error) {
	proof, err := p.smTree.Prove(key)
	if err != nil {
		return nil, err
	}

	return encodeProof(&proof)
}

type stateProofDB struct {
	backend backend.Backend
}

func (s *stateProofDB) Get(key []byte) ([]byte, error) {
	lastVer := s.backend.LastVersion()
	return s.backend.Get(key, &lastVer)
}

func (s *stateProofDB) Set(key []byte, value []byte) error {
	return s.backend.Set(key, value)
}

func (s *stateProofDB) Delete(key []byte) error {
	return s.backend.Delete(key)
}

func VerifyProof(proofData []byte, root []byte, key []byte, value []byte) bool {
	proof, err := decodeProof(proofData)
	if err != nil {
		return false
	}

	return smt.VerifyProof(*proof, root, key, value, sha256.New())
}
