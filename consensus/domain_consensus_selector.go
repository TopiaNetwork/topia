package consensus

import (
	"math/big"

	tpchaintypes "github.com/TopiaNetwork/topia/chain/types"
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

func (selector *domainConsensusSelector) Select(selfNodeID string, compState state.CompositionState) (string, DKGBls, []*tpcmm.NodeDomainMember, *tpcmm.NodeDomainMember, error) {
	var selfNode *tpcmm.NodeInfo
	var latestBlock *tpchaintypes.Block
	var activeCSDomains []*tpcmm.NodeDomainInfo

	if compState.CompSState() == state.CompSState_Commited {
		compStateRN := state.CreateCompositionStateReadonly(selector.log, selector.ledger)
		//selector.log.Infof("Fetched read-only composition state version: %d, number %d, self node %s", compState.StateVersion(), len(activeCSDomains), selfNodeID)

		selfNodeT, err := compStateRN.GetNode(selfNodeID)
		if err != nil {
			return "", nil, nil, nil, err
		}
		selfNode = selfNodeT

		latestBlock, err = compStateRN.GetLatestBlock()
		if err != nil {
			return "", nil, nil, nil, err
		}

		activeCSDomains, err = compStateRN.GetAllActiveNodeConsensusDomains(latestBlock.Head.Height)
		if err != nil {
			return "", nil, nil, nil, err
		}
	} else {
		//selector.log.Infof("Fetched top state version: %d, number %d, self node %s", compState.StateVersion(), len(activeCSDomains), selfNodeID)
		selfNodeT, err := compState.GetNode(selfNodeID)
		if err != nil {
			return "", nil, nil, nil, err
		}
		selfNode = selfNodeT

		latestBlock, err = compState.GetLatestBlock()
		if err != nil {
			return "", nil, nil, nil, err
		}

		activeCSDomains, err = compState.GetAllActiveNodeConsensusDomains(latestBlock.Head.Height + 1)
		if err != nil {
			return "", nil, nil, nil, err
		}
	}
	if len(activeCSDomains) == 0 {
		//selector.log.Warn("No available consensus domain at present")
		return "", nil, nil, nil, nil
	}

	selector.log.Infof("Get available consensus domains: number %d, self node %s", len(activeCSDomains), selfNodeID)

	hasher := tpcmm.NewBlake2bHasher(0)
	hashBytes := hasher.Compute(string(latestBlock.Head.VRFProof))
	hashBig := new(big.Int).SetBytes(hashBytes[:])

	index := hashBig.Mod(hashBig, big.NewInt(int64(len(activeCSDomains)))).Int64()

	selectedCSDomain := activeCSDomains[index]

	if selfNode.Role&tpcmm.NodeRole_Executor == tpcmm.NodeRole_Executor {
		return selectedCSDomain.ID, nil, selectedCSDomain.CSDomainData.Members, nil, nil
	} else {
		domainCSCrypt, selfSelected := NewDomainConsensusCrypt(selfNode, selectedCSDomain)
		return selectedCSDomain.ID, domainCSCrypt, selectedCSDomain.CSDomainData.Members, selfSelected, nil
	}
}
