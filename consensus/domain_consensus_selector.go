package consensus

import (
	"errors"
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

func (selector *domainConsensusSelector) Select(selfNodeID string, stateBuilderType state.CompStateBuilderType) (DKGBls, []*tpcmm.NodeDomainMember, uint32, error) {
	compState := state.GetStateBuilder(stateBuilderType).TopCompositionState(selfNodeID)
	if compState == nil {
		return nil, nil, 0, errors.New("Top composition state nil")
	}

	selfNode, err := compState.GetNode(selfNodeID)
	if err != nil {
		return nil, nil, 0, err
	}

	latestBlock, err := compState.GetLatestBlock()
	if err != nil {
		return nil, nil, 0, err
	}

	activeCSDomains, err := compState.GetAllActiveNodeConsensusDomains(latestBlock.Head.Height)
	if err != nil {
		return nil, nil, 0, err
	}
	if len(activeCSDomains) == 0 {
		//selector.log.Warn("No available consensus domain at present")
		return nil, nil, 0, nil
	}

	selector.log.Infof("Get available consensus domains: number %d", len(activeCSDomains))

	hasher := tpcmm.NewBlake2bHasher(0)
	hashBytes := hasher.Compute(string(latestBlock.Head.VRFProof))
	hashBig := new(big.Int).SetBytes(hashBytes[:])

	index := hashBig.Mod(hashBig, big.NewInt(int64(len(activeCSDomains)))).Int64()

	selectedCSDomain := activeCSDomains[index]

	domainCSCrypt, selfSelected := NewDomainConsensusCrypt(selfNode, selectedCSDomain)

	return domainCSCrypt, selectedCSDomain.CSDomainData.Members, selfSelected, nil
}
