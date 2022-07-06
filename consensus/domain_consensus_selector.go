package consensus

import (
	"math/big"

	tpcmm "github.com/TopiaNetwork/topia/common"
	"github.com/TopiaNetwork/topia/ledger"
	tplog "github.com/TopiaNetwork/topia/log"
	"github.com/TopiaNetwork/topia/state"
)

type domainConsensusSelector struct {
	log    tplog.Logger
	ledger ledger.Ledger
}

func NewDomainConsensusSelector(log tplog.Logger, ledger ledger.Ledger) *domainConsensusSelector {
	return &domainConsensusSelector{
		log:    log,
		ledger: ledger,
	}
}

func (selector *domainConsensusSelector) Select(selfNodeID string) (DKGBls, []*tpcmm.NodeDomainMember, uint32, error) {
	compStateRN := state.CreateCompositionStateReadonly(selector.log, selector.ledger)

	selfNode, err := compStateRN.GetNode(selfNodeID)
	if err != nil {
		return nil, nil, 0, err
	}

	latestBlock, err := compStateRN.GetLatestBlock()
	if err != nil {
		return nil, nil, 0, err
	}

	activeCSDomains, err := compStateRN.GetAllActiveNodeConsensusDomains(latestBlock.Head.Height)
	if err != nil {
		return nil, nil, 0, err
	}
	if len(activeCSDomains) == 0 {
		selector.log.Warn("No available consensus domain at present")
		return nil, nil, 0, nil
	}

	hasher := tpcmm.NewBlake2bHasher(0)
	hashBytes := hasher.Compute(string(latestBlock.Head.VRFProof))
	hashBig := new(big.Int).SetBytes(hashBytes[:])

	index := hashBig.Mod(hashBig, big.NewInt(int64(len(activeCSDomains)))).Int64()

	selectedCSDomain := activeCSDomains[index]

	domainCSCrypt, selfSelected := NewDomainConsensusCrypt(selfNode, selectedCSDomain)

	return domainCSCrypt, selectedCSDomain.CSDomainData.Members, selfSelected, nil
}
