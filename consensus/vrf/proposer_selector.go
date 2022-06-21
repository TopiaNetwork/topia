package vrf

import (
	"fmt"
	"math/big"

	tpcrt "github.com/TopiaNetwork/topia/crypt"
	tpcrtypes "github.com/TopiaNetwork/topia/crypt/types"
)

type ProposerSelectionType int

const (
	ProposerSelectionType_Unknown = iota
	ProposerSelectionType_Poiss
)

type ProposerSelector interface {
	ComputeVRF(priKey tpcrtypes.PrivateKey, data []byte) ([]byte, error)

	VerifyVRF(addr tpcrtypes.Address, data []byte, vrfProof []byte) (bool, error)

	SelectProposer(VRFProof []byte, weight *big.Int, totalWeight *big.Int) int64

	MaxPriority(vrf []byte, winCount int64) []byte
}

func NewProposerSelector(psType ProposerSelectionType, crypt tpcrt.CryptService) ProposerSelector {
	switch psType {
	case ProposerSelectionType_Poiss:
		return newProposerSelectorPoiss(crypt)
	default:
		panic(fmt.Sprintf("invalid proposer selector: %d", psType))
	}

	return nil
}
