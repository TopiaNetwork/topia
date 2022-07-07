package consensus

import (
	"context"
	"fmt"
	"github.com/TopiaNetwork/topia/eventhub"
	"sync"
	"sync/atomic"
	"time"

	tpchaintypes "github.com/TopiaNetwork/topia/chain/types"
	tpcmm "github.com/TopiaNetwork/topia/common"
	"github.com/TopiaNetwork/topia/execution"
	"github.com/TopiaNetwork/topia/ledger"
	tplog "github.com/TopiaNetwork/topia/log"
	"github.com/TopiaNetwork/topia/state"
)

type EpochService interface {
	GetActiveExecutorIDs() []string

	GetActiveProposerIDs() []string

	GetActiveValidatorIDs() []string

	GetNodeWeight(nodeID string) (uint64, error)

	GetActiveExecutorsTotalWeight() uint64

	GetActiveProposersTotalWeight() uint64

	GetActiveValidatorsTotalWeight() uint64

	GetLatestEpoch() *tpcmm.EpochInfo

	SelfSelected() bool

	AddDKGBLSUpdater(updater DKGBLSUpdater)

	UpdateEpoch(ctx context.Context, newBH *tpchaintypes.BlockHead, compState state.CompositionState) error

	Start(ctx context.Context)
}

type activeNodeInfos struct {
	sync                   sync.RWMutex
	activeExecutorIDs      []string
	activeProposerIDs      []string
	activeValidatorIDs     []string
	activeExecutorWeights  uint64
	activeProposerWeights  uint64
	activeValidatorWeights uint64
	nodeWeights            map[string]uint64 //nodeID->weight
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
	an.activeExecutorIDs = an.activeValidatorIDs[:0]

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

	if len(an.activeProposerIDs) != 0 {
		panic("")
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

		dkgCPT, members, selfSelected, err := es.csDomainSelector.Select(es.nodeID, es.stateBuilderType)
		if err != nil {
			es.log.Errorf("Select consensus domain error: %v", err)
			return err
		}
		if dkgCPT != nil {
			es.activeNodeInfos.update(es.log, es.ledger, members)
			es.notifyUpdater(dkgCPT)
			atomic.StoreUint32(&es.selfSelected, selfSelected)
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

func (es *epochService) Start(ctx context.Context) {
	go func() {
		for {
			dkgCPT, members, selfSelected, err := es.csDomainSelector.Select(es.nodeID, es.stateBuilderType)
			if err != nil {
				es.log.Errorf("Select consensus domain error: %v", err)
				continue
			}
			if dkgCPT != nil {
				es.activeNodeInfos.update(es.log, es.ledger, members)
				es.notifyUpdater(dkgCPT)
				atomic.StoreUint32(&es.selfSelected, selfSelected)
				return
			} else {
				time.Sleep(50 * time.Millisecond)
			}
		}
	}()
}
