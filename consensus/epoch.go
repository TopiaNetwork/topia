package consensus

import (
	"context"
	"fmt"
	"sync"
	"sync/atomic"
	"time"

	tpchaintypes "github.com/TopiaNetwork/topia/chain/types"
	tpcmm "github.com/TopiaNetwork/topia/common"
	"github.com/TopiaNetwork/topia/eventhub"
	"github.com/TopiaNetwork/topia/execution"
	"github.com/TopiaNetwork/topia/ledger"
	tplog "github.com/TopiaNetwork/topia/log"
	"github.com/TopiaNetwork/topia/state"
)

type EpochService interface {
	GetActiveExecutorIDs() []string

	GetActiveExecutorIDsOfDomain(domainID string) []string

	GetDomainIDOfExecutor(nodeID string) string

	GetActiveProposerIDs() []string

	GetActiveValidatorIDs() []string

	GetNodeWeight(nodeID string) (uint64, error)

	GetActiveExecutorsTotalWeight() uint64

	GetActiveExecutorsTotalWeightOfDomain(domainID string) uint64

	GetActiveProposersTotalWeight() uint64

	GetActiveValidatorsTotalWeight() uint64

	GetLatestEpoch() *tpcmm.EpochInfo

	SelfSelected() bool

	AddDKGBLSUpdater(updater DKGBLSUpdater)

	UpdateData(log tplog.Logger, ledger ledger.Ledger, csDomainMember []*tpcmm.NodeDomainMember)

	UpdateEpoch(ctx context.Context, newBH *tpchaintypes.BlockHead, compState state.CompositionState) error

	Start(ctx context.Context, latestHeight uint64)
}

type activeNodeInfos struct {
	sync                          sync.RWMutex
	activeExecutorIDs             []string
	activeExecutorIDsOfDomain     map[string][]string
	activeProposerIDs             []string
	activeValidatorIDs            []string
	activeExecutorWeights         uint64
	activeExecutorWeightsOfDomain map[string]uint64
	activeProposerWeights         uint64
	activeValidatorWeights        uint64
	nodeWeights                   map[string]uint64 //nodeID->weight
}

type epochService struct {
	log                 tplog.Logger
	nodeID              string
	stateBuilderType    state.CompStateBuilderType
	epochInterval       uint64 //the height number between two epochs
	currentEpoch        *tpcmm.EpochInfo
	dkgStartBeforeEpoch uint64 //the starting height number of DKG before an epoch
	exeScheduler        execution.ExecutionScheduler
	ledger              ledger.Ledger
	deliver             messageDeliverI
	dkgExchange         *dkgExchange
	csDomainSelector    *domainConsensusSelector
	activeNodeInfos     *activeNodeInfos
	selfSelected        uint32
	updatersSync        sync.RWMutex
	dkgBLSUpdaters      []DKGBLSUpdater
}

func NewEpochService(log tplog.Logger,
	nodeID string,
	stateBuilderType state.CompStateBuilderType,
	epochInterval uint64,
	currentEpoch *tpcmm.EpochInfo,
	dkgStartBeforeEpoch uint64,
	exeScheduler execution.ExecutionScheduler,
	ledger ledger.Ledger,
	deliver messageDeliverI,
	dkgExchange *dkgExchange,
	exeActiveNodeIds []*tpcmm.NodeInfo) EpochService {
	if ledger == nil || dkgExchange == nil {
		log.Panic("Invalid input parameter and can't create epoch service!")
	}

	anInfos := &activeNodeInfos{}
	anInfos.nodeWeights = make(map[string]uint64)
	for _, executorInfo := range exeActiveNodeIds {
		anInfos.activeExecutorIDs = append(anInfos.activeExecutorIDs, executorInfo.NodeID)
		anInfos.activeExecutorWeights += executorInfo.Weight
		anInfos.nodeWeights[executorInfo.NodeID] = executorInfo.Weight
	}

	csDomainSel := NewDomainConsensusSelector(log, ledger)

	epochS := &epochService{
		log:                 log,
		nodeID:              nodeID,
		stateBuilderType:    stateBuilderType,
		epochInterval:       epochInterval,
		currentEpoch:        currentEpoch,
		dkgStartBeforeEpoch: dkgStartBeforeEpoch,
		exeScheduler:        exeScheduler,
		ledger:              ledger,
		deliver:             deliver,
		dkgExchange:         dkgExchange,
		csDomainSelector:    csDomainSel,
		activeNodeInfos:     anInfos,
	}

	exeScheduler.SetEpochUpdater(epochS)

	return epochS
}

func (an *activeNodeInfos) update(log tplog.Logger, ledger ledger.Ledger, csDomainMember []*tpcmm.NodeDomainMember) {
	an.sync.Lock()
	defer an.sync.Unlock()

	compStateRN := state.CreateCompositionStateReadonly(log, ledger)

	an.activeExecutorWeights = 0
	an.activeProposerWeights = 0
	an.activeValidatorWeights = 0

	an.activeExecutorIDs = an.activeExecutorIDs[:0]
	an.activeProposerIDs = an.activeProposerIDs[:0]
	an.activeValidatorIDs = an.activeValidatorIDs[:0]

	an.activeExecutorIDsOfDomain = make(map[string][]string)
	an.activeExecutorWeightsOfDomain = make(map[string]uint64)
	an.nodeWeights = make(map[string]uint64)

	executorInfos, err := compStateRN.GetAllActiveExecutors()
	if err != nil {
		log.Errorf("Get active executor infos failed: %v", err)
		return
	}
	for _, executorInfo := range executorInfos {
		an.activeExecutorIDs = append(an.activeExecutorIDs, executorInfo.NodeID)
		an.activeExecutorWeights += executorInfo.Weight
		an.nodeWeights[executorInfo.NodeID] = executorInfo.Weight
	}

	latestBlock, _ := compStateRN.GetLatestBlock()
	executorDomainInfos, err := compStateRN.GetAllActiveNodeExecuteDomains(latestBlock.Head.Height)
	if err != nil {
		log.Errorf("Get active executor domain infos failed: %v", err)
		return
	}
	for _, exeDomainInfo := range executorDomainInfos {
		an.activeExecutorIDsOfDomain[exeDomainInfo.ID] = append(an.activeExecutorIDsOfDomain[exeDomainInfo.ID], exeDomainInfo.ExeDomainData.Members...)
		for _, exeID := range exeDomainInfo.ExeDomainData.Members {
			an.activeExecutorWeightsOfDomain[exeDomainInfo.ID] += an.nodeWeights[exeID]
		}
	}

	for _, domainMember := range csDomainMember {
		if domainMember.NodeRole&tpcmm.NodeRole_Proposer == tpcmm.NodeRole_Proposer {
			an.activeProposerIDs = append(an.activeProposerIDs, domainMember.NodeID)
			an.activeProposerWeights += domainMember.Weight
		} else if domainMember.NodeRole&tpcmm.NodeRole_Validator == tpcmm.NodeRole_Validator {
			an.activeValidatorIDs = append(an.activeValidatorIDs, domainMember.NodeID)
			an.activeValidatorWeights += domainMember.Weight
		}
		an.nodeWeights[domainMember.NodeID] = domainMember.Weight
	}
}

func (an *activeNodeInfos) getActiveExecutorIDs() []string {
	an.sync.RLock()
	defer an.sync.RUnlock()

	return an.activeExecutorIDs
}

func (an *activeNodeInfos) getActiveExecutorIDsOfDomain(domainID string) []string {
	an.sync.RLock()
	defer an.sync.RUnlock()

	if exeIDs, ok := an.activeExecutorIDsOfDomain[domainID]; ok {
		return exeIDs
	}

	return nil
}

func (an *activeNodeInfos) getDomainIDOfExecutor(nodeID string) string {
	an.sync.RLock()
	defer an.sync.RUnlock()

	for domainID, nodeIDs := range an.activeExecutorIDsOfDomain {
		if tpcmm.IsContainString(nodeID, nodeIDs) {
			return domainID
		}
	}

	return ""
}

func (an *activeNodeInfos) getActiveProposerIDs() []string {
	an.sync.RLock()
	defer an.sync.RUnlock()

	return an.activeProposerIDs
}

func (an *activeNodeInfos) getActiveValidatorIDs() []string {
	an.sync.RLock()
	defer an.sync.RUnlock()

	return an.activeValidatorIDs
}

func (an *activeNodeInfos) getNodeWeight(nodeID string) (uint64, error) {
	an.sync.RLock()
	defer an.sync.RUnlock()

	if weight, ok := an.nodeWeights[nodeID]; ok {
		return weight, nil
	}

	return 0, fmt.Errorf("Can't get node weight from epoch service: %s", nodeID)
}

func (an *activeNodeInfos) getActiveExecutorsTotalWeight() uint64 {
	an.sync.RLock()
	defer an.sync.RUnlock()

	return an.activeExecutorWeights
}

func (an *activeNodeInfos) getActiveExecutorsTotalWeightOfDomain(domainID string) uint64 {
	an.sync.RLock()
	defer an.sync.RUnlock()

	if weight, ok := an.activeExecutorWeightsOfDomain[domainID]; ok {
		return weight
	}

	return 0
}

func (an *activeNodeInfos) getActiveProposersTotalWeight() uint64 {
	an.sync.RLock()
	defer an.sync.RUnlock()

	return an.activeProposerWeights
}

func (an *activeNodeInfos) getActiveValidatorsTotalWeight() uint64 {
	an.sync.RLock()
	defer an.sync.RUnlock()

	return an.activeValidatorWeights
}

func (es *epochService) GetActiveExecutorIDs() []string {
	return es.activeNodeInfos.getActiveExecutorIDs()
}

func (es *epochService) GetActiveExecutorIDsOfDomain(domainID string) []string {
	return es.activeNodeInfos.getActiveExecutorIDsOfDomain(domainID)
}

func (es *epochService) GetDomainIDOfExecutor(nodeID string) string {
	return es.activeNodeInfos.getDomainIDOfExecutor(nodeID)
}

func (es *epochService) GetActiveProposerIDs() []string {
	return es.activeNodeInfos.getActiveProposerIDs()
}

func (es *epochService) GetActiveValidatorIDs() []string {
	return es.activeNodeInfos.getActiveValidatorIDs()
}

func (es *epochService) GetNodeWeight(nodeID string) (uint64, error) {
	return es.activeNodeInfos.getNodeWeight(nodeID)
}

func (es *epochService) GetActiveExecutorsTotalWeight() uint64 {
	return es.activeNodeInfos.getActiveExecutorsTotalWeight()
}

func (es *epochService) GetActiveExecutorsTotalWeightOfDomain(domainID string) uint64 {
	return es.activeNodeInfos.getActiveExecutorsTotalWeightOfDomain(domainID)
}

func (es *epochService) GetActiveProposersTotalWeight() uint64 {
	return es.activeNodeInfos.getActiveProposersTotalWeight()
}

func (es *epochService) GetActiveValidatorsTotalWeight() uint64 {
	return es.activeNodeInfos.getActiveValidatorsTotalWeight()
}

func (es *epochService) GetLatestEpoch() *tpcmm.EpochInfo {
	return es.currentEpoch
}

func (es *epochService) SelfSelected() bool {
	return atomic.LoadUint32(&es.selfSelected) == 1
}

func (es *epochService) UpdateData(log tplog.Logger, ledger ledger.Ledger, csDomainMember []*tpcmm.NodeDomainMember) {
	es.activeNodeInfos.update(log, ledger, csDomainMember)
	es.deliver.deliverNetwork().UpdateNetActiveNode(es)
}

func (es *epochService) UpdateEpoch(ctx context.Context, newBH *tpchaintypes.BlockHead, compState state.CompositionState) error {
	deltaH := int(newBH.Height) - int(es.currentEpoch.StartHeight)
	if deltaH == int(es.epochInterval) {
		newEpoch := &tpcmm.EpochInfo{
			Epoch:          es.currentEpoch.Epoch + 1,
			StartTimeStamp: uint64(time.Now().UnixNano()),
			StartHeight:    newBH.Height,
		}
		err := compState.SetLatestEpoch(newEpoch)
		if err != nil {
			es.log.Errorf("Save the latest epoch error: %v", err)
			return err
		}

		domainID, dkgCPT, members, selfSelected, err := es.csDomainSelector.Select(es.nodeID, compState)
		if err != nil {
			es.log.Errorf("Select consensus domain error: %v", err)
			return err
		}
		if dkgCPT != nil {
			es.UpdateData(es.log, es.ledger, members)
			es.notifyUpdater(dkgCPT)
			if selfSelected != nil {
				atomic.StoreUint32(&es.selfSelected, 1)

				csDomainSelectedMsg := &ConsensusDomainSelectedMessage{
					ChainID:            []byte(compState.ChainID()),
					Version:            CONSENSUS_VER,
					DomainID:           []byte(domainID),
					MemberNumber:       uint32(len(members)),
					NodeIDOfMember:     []byte(selfSelected.NodeID),
					NodeRoleOfMember:   uint64(selfSelected.NodeRole),
					NodeWeightOfMember: selfSelected.Weight,
				}
				err = es.deliver.deliverConsensusDomainSelectedMessageExe(ctx, csDomainSelectedMsg)
				if err == nil {
					es.log.Infof("Successfully deliver consensus domain selected message to execute network")
				}
			} else {
				atomic.StoreUint32(&es.selfSelected, 0)
			}
		}

		eventhub.GetEventHubManager().GetEventHub(es.nodeID).Trig(ctx, eventhub.EventName_EpochNew, newEpoch)
	}

	return nil
}

func (es *epochService) AddDKGBLSUpdater(updater DKGBLSUpdater) {
	es.updatersSync.Lock()
	defer es.updatersSync.Unlock()

	es.dkgBLSUpdaters = append(es.dkgBLSUpdaters, updater)
}

func (es *epochService) notifyUpdater(dkgBls DKGBls) {
	es.updatersSync.RLock()
	defer es.updatersSync.RUnlock()
	for _, updater := range es.dkgBLSUpdaters {
		updater.updateDKGBls(dkgBls)
	}
}

func (es *epochService) Start(ctx context.Context, latestHeight uint64) {
	go func() {
		for {
			var compState state.CompositionState
			switch es.stateBuilderType {
			case state.CompStateBuilderType_Full:
				compState = es.exeScheduler.CompositionStateAtVersion(latestHeight + 1)
			case state.CompStateBuilderType_Simple:
				compState = state.GetStateBuilder(es.stateBuilderType).CompositionStateAtVersion(es.nodeID, latestHeight+1)
			}

			if compState == nil {
				time.Sleep(50 * time.Millisecond)
				continue
			}

			//es.log.Infof("Fetched composition state: state version %d, state %d, self node %s", compState.StateVersion(), compState.CompSState(), es.nodeID)

			domainID, dkgCPT, members, selfSelected, err := es.csDomainSelector.Select(es.nodeID, compState)
			if err != nil {
				es.log.Errorf("Select consensus domain error: %v", err)
				continue
			}
			if len(members) > 0 && dkgCPT != nil {
				es.UpdateData(es.log, es.ledger, members)
				es.notifyUpdater(dkgCPT)
				if selfSelected != nil {
					atomic.StoreUint32(&es.selfSelected, 1)

					csDomainSelectedMsg := &ConsensusDomainSelectedMessage{
						ChainID:            []byte(compState.ChainID()),
						Version:            CONSENSUS_VER,
						DomainID:           []byte(domainID),
						MemberNumber:       uint32(len(members)),
						NodeIDOfMember:     []byte(selfSelected.NodeID),
						NodeRoleOfMember:   uint64(selfSelected.NodeRole),
						NodeWeightOfMember: selfSelected.Weight,
					}
					err = es.deliver.deliverConsensusDomainSelectedMessageExe(ctx, csDomainSelectedMsg)
					if err == nil {
						es.log.Infof("Successfully deliver consensus domain selected message to execute network: self node %s", es.nodeID)
					}
				} else {
					atomic.StoreUint32(&es.selfSelected, 0)
				}

				return
			} else {
				time.Sleep(50 * time.Millisecond)
			}
		}
	}()
}
