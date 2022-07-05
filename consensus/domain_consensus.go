package consensus

import (
	"context"
	"fmt"
	"math"

	tpchaintypes "github.com/TopiaNetwork/topia/chain/types"
	tpcmm "github.com/TopiaNetwork/topia/common"
	"github.com/TopiaNetwork/topia/configuration"
	"github.com/TopiaNetwork/topia/consensus/vrf"
	"github.com/TopiaNetwork/topia/ledger"
	tplog "github.com/TopiaNetwork/topia/log"
	"github.com/TopiaNetwork/topia/state"
)

const (
	ConsensusDomain_MinProposer             = 3
	ConsensusDomain_MaxProposer             = 30
	ConsensusDomain_ProposerRadio           = 0.3
	ConsensusDomain_MinValidator            = 4
	ConsensusDomain_MaxValidator            = 70
	ConsensusDomain_MaxDomainOfEachNode     = 5 //Max Domain count that each node can join
	ConsensusDomain_TriggerTimesOfEachEpoch = 1
)

type domainConsensusService struct {
	nodeID         string
	log            tplog.Logger
	ledger         ledger.Ledger
	blockAddedCh   chan *tpchaintypes.Block
	selector       vrf.RoleSelectorVRF
	csConfig       *configuration.ConsensusConfiguration
	triggerNumber  uint64
	triggerBlock   *tpchaintypes.Block
	selfSelected   bool
	candidateNodes []*tpcmm.NodeDomainMember
	dkgEx          *dkgExchange
}

func NewDomainConsensusService(
	nodeID string,
	log tplog.Logger,
	ledger ledger.Ledger,
	blockAddedCh chan *tpchaintypes.Block,
	selector vrf.RoleSelectorVRF,
	csConfig *configuration.ConsensusConfiguration,
	dkgEx *dkgExchange) *domainConsensusService {

	return &domainConsensusService{
		nodeID:       nodeID,
		log:          log,
		ledger:       ledger,
		blockAddedCh: blockAddedCh,
		selector:     selector,
		csConfig:     csConfig,
		dkgEx:        dkgEx,
	}
}

func (ds *domainConsensusService) removeInvalidActiveNodes(activeNodes []*tpcmm.NodeInfo, nodeDomains []*tpcmm.NodeDomainInfo) []*tpcmm.NodeInfo {
	var goodActiveNodes []*tpcmm.NodeInfo

	for _, nodeInfo := range activeNodes {
		joinedDomainCnt := 0
		isValid := func() bool {
			for _, ndInfo := range nodeDomains {
				if tpcmm.IsContainString(nodeInfo.NodeID, tpcmm.NodeIDs(ndInfo.CSDomainData.Members)) {
					joinedDomainCnt++
				}

				if joinedDomainCnt > ConsensusDomain_MaxDomainOfEachNode {
					return false
				}
			}

			return true
		}()

		if isValid {
			goodActiveNodes = append(goodActiveNodes, nodeInfo)
		}
	}

	return goodActiveNodes
}

func (ds *domainConsensusService) getJoinableConsensusCandidateNodes(blk *tpchaintypes.Block) ([]*tpcmm.NodeInfo, []*tpcmm.NodeInfo, error) {
	compStateRN := state.CreateCompositionStateReadonly(ds.log, ds.ledger)
	defer compStateRN.Stop()

	epochInfo, err := compStateRN.GetLatestEpoch()
	if err != nil {
		return nil, nil, err
	}

	activeProposers, err := compStateRN.GetAllActiveProposers()
	if err != nil {
		return nil, nil, err
	}
	activePropCnt := len(activeProposers)
	if activePropCnt < ConsensusDomain_MinProposer {
		return nil, nil, fmt.Errorf("Not enough active proposer: %d, required min count %d", activePropCnt, ConsensusDomain_MinProposer)
	}

	activeValidators, err := compStateRN.GetAllActiveValidators()
	if err != nil {
		return nil, nil, err
	}
	activeValCnt := len(activeValidators)
	if activeValCnt < ConsensusDomain_MinValidator {
		return nil, nil, fmt.Errorf("Not enough active validator: %d, required min count %d", activeValCnt, ConsensusDomain_MinValidator)
	}

	nodeDomains, err := compStateRN.GetAllActiveNodeConsensusDomains(blk.Head.Height)
	if err != nil {
		return nil, nil, err
	}

	goodActivePropNodes := ds.removeInvalidActiveNodes(activeProposers, nodeDomains)
	goodActivePropCnt := len(goodActivePropNodes)
	if goodActivePropCnt < ConsensusDomain_MinProposer {
		return nil, nil, fmt.Errorf("Not enough active proposer: %d, required min count %d", activePropCnt, ConsensusDomain_MinProposer)
	}
	if goodActivePropCnt > ConsensusDomain_MaxProposer {
		goodActivePropCnt = ConsensusDomain_MaxProposer
		goodActivePropNodes, err = ds.selector.SelectExpectedNodes(epochInfo, blk, goodActivePropNodes, goodActivePropCnt)
		if err != nil {
			return nil, nil, err
		}
	}

	goodActiveValNodes := ds.removeInvalidActiveNodes(activeValidators, nodeDomains)
	goodActiveValCnt := len(goodActiveValNodes)
	if goodActiveValCnt < ConsensusDomain_MinValidator {
		return nil, nil, fmt.Errorf("Not enough active validator: %d, required min count %d", activeValCnt, ConsensusDomain_MinValidator)
	}
	if goodActiveValCnt > ConsensusDomain_MaxValidator {
		goodActiveValCnt = ConsensusDomain_MaxProposer
		goodActiveValNodes, err = ds.selector.SelectExpectedNodes(epochInfo, blk, goodActiveValNodes, goodActiveValCnt)
		if err != nil {
			return nil, nil, err
		}
	}

	pvRadio := float64(goodActivePropCnt) / float64(goodActiveValCnt)
	if pvRadio > ConsensusDomain_ProposerRadio && goodActivePropCnt > ConsensusDomain_MinProposer {
		goodActivePropCnt = int(math.Ceil(ConsensusDomain_ProposerRadio * float64(goodActiveValCnt)))
		if goodActivePropCnt < ConsensusDomain_MinProposer {
			goodActivePropCnt = ConsensusDomain_MinProposer
		}
		goodActivePropNodes, err = ds.selector.SelectExpectedNodes(epochInfo, blk, goodActivePropNodes, goodActivePropCnt)
		if err != nil {
			return nil, nil, err
		}
	} else if pvRadio < ConsensusDomain_ProposerRadio && goodActiveValCnt > ConsensusDomain_MinValidator {
		goodActiveValCnt = int(math.Ceil((1.0 - ConsensusDomain_ProposerRadio) * float64(goodActiveValCnt)))
		if goodActiveValCnt < ConsensusDomain_MinValidator {
			goodActiveValCnt = ConsensusDomain_MinValidator
		}
		goodActiveValNodes, err = ds.selector.SelectExpectedNodes(epochInfo, blk, goodActiveValNodes, goodActiveValCnt)
		if err != nil {
			return nil, nil, err
		}
	}

	var googCSNodes []*tpcmm.NodeInfo
	googCSNodes = append(googCSNodes, goodActivePropNodes...)
	googCSNodes = append(googCSNodes, goodActiveValNodes...)

	return goodActivePropNodes, goodActiveValNodes, nil
}

func (ds *domainConsensusService) collectConsensusCandidateNodeStart(ctx context.Context) {
	go func() {
		for {
			select {
			case newBlock := <-ds.blockAddedCh:
				ds.log.Infof("Domain service received new block: height %d, self node %s", newBlock.Head.Height, ds.nodeID)

				if ds.dkgEx.getDKGState() != DKGExchangeState_IDLE {
					ds.log.Warnf("DKG exchange is busy: height %d, self node %s", newBlock.Head.Height, ds.nodeID)
					continue
				}

				tSpan := ds.csConfig.BlocksPerEpoch / ConsensusDomain_TriggerTimesOfEachEpoch
				tNumber := newBlock.Head.Height / tSpan * tSpan
				if tNumber == ds.triggerNumber {
					continue
				}
				ds.triggerNumber = tNumber
				ds.triggerBlock = newBlock
				ds.log.Infof("Trigger new candidate nodes section: height %d, trigger round %d, self node %s", newBlock.Head.Height, ds.triggerNumber, ds.nodeID)

				propCandidateNodes, valCandidateNodes, err := ds.getJoinableConsensusCandidateNodes(newBlock)
				if err != nil {
					ds.log.Errorf("Can't get joinable candidate nodes: %v, self node %s", err, ds.nodeID)
					continue
				}

				dkgPartKeyPubs := make(map[string]string, len(propCandidateNodes)+len(valCandidateNodes))

				var propCandNodeIDs []string
				var valCandNodeIDs []string
				for _, propCandNode := range propCandidateNodes {
					if propCandNode.NodeID == ds.nodeID {
						ds.selfSelected = true
					} else {
						propCandNodeIDs = append(propCandNodeIDs, propCandNode.NodeID)
					}

					ds.candidateNodes = append(ds.candidateNodes, &tpcmm.NodeDomainMember{NodeID: propCandNode.NodeID, NodeRole: tpcmm.NodeRole_Proposer, Weight: propCandNode.Weight})

					dkgPartKeyPubs[propCandNode.NodeID] = propCandNode.DKGPartPubKey
				}
				for _, valCandNode := range valCandidateNodes {
					if valCandNode.NodeID == ds.nodeID {
						ds.selfSelected = true
					} else {
						valCandNodeIDs = append(valCandNodeIDs, valCandNode.NodeID)
					}

					ds.candidateNodes = append(ds.candidateNodes, &tpcmm.NodeDomainMember{NodeID: valCandNode.NodeID, NodeRole: tpcmm.NodeRole_Validator, Weight: valCandNode.Weight})

					dkgPartKeyPubs[valCandNode.NodeID] = valCandNode.DKGPartPubKey
				}

				if ds.selfSelected {
					ds.dkgEx.updateDKGPartPubKeys(dkgPartKeyPubs)
					ds.dkgEx.deliver.updateCandNodeIDs(propCandNodeIDs, valCandNodeIDs)
					ds.dkgEx.initWhenStart(ds.triggerNumber)
					ds.dkgEx.addDKGBLSUpdater(ds)
					ds.dkgEx.start(ds.triggerNumber)
				}
			case <-ctx.Done():
				ds.log.Infof("Collect consensus candidate nodes exit: self node %s", ds.nodeID)
			}
		}
	}()
}

func (ds *domainConsensusService) updateDKGBls(dkgBls DKGBls) {
	ndInfo := &tpcmm.NodeDomainInfo{}

	bHash, _ := ds.triggerBlock.BlockHash()
	ndInfo.ID = string(bHash)
	ndInfo.Type = tpcmm.DomainType_Consensus

	if ds.triggerNumber == 1 {
		ndInfo.ValidHeightStart = ds.triggerBlock.Head.Height + 1
	}
	ndInfo.ValidHeightEnd = ndInfo.ValidHeightStart + 50*tpcmm.EpochSpan

	pubKey, _ := dkgBls.PubKey()
	priShare, _ := dkgBls.PriShare()
	pubShares, _ := dkgBls.PubShares()

	csDomainData := &tpcmm.NodeConsensusDomain{
		Threshold:    dkgBls.Threshold(),
		NParticipant: len(ds.candidateNodes),
		PublicKey:    pubKey,
		PubShares:    pubShares,
		Members:      ds.candidateNodes,
	}

	ndInfo.CSDomainData = csDomainData

	compState := state.GetStateBuilder(state.CompStateBuilderType_Simple).TopCompositionState(ds.nodeID)
	if compState == nil {
		return
	}

	compState.UpdateDKGPriShare(ds.nodeID, priShare)
	compState.AddNodeDomain(ndInfo)
}

func (ds domainConsensusService) Trigger(newBlockAdded *tpchaintypes.Block) {
	ds.blockAddedCh <- newBlockAdded
}

func (ds *domainConsensusService) Start(ctx context.Context) {
	ds.collectConsensusCandidateNodeStart(ctx)
}
