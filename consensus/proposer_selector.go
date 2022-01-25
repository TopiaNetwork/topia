package consensus

import (
	"fmt"
	"math/big"

	tpcmm "github.com/TopiaNetwork/topia/common"
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

	SelectProposer(VRFProof []byte, weight *big.Int, totalWeight *big.Int) int64
}

func NewProposerSelector(psType ProposerSelectionType, crypt tpcrt.CryptService, hasher tpcmm.Hasher) ProposerSelector {
	switch psType {
	case ProposerSelectionType_Poiss:
		return newProposerSelectorPoiss(crypt, hasher)
	default:
		panic(fmt.Sprintf("invalid proposer selector: %d", psType))
	}

	return nil
}
