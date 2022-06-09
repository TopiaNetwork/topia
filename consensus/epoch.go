package consensus

import (
	"context"
	"fmt"
	"sync"

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
	blockAddedCh        chan *tpchaintypes.Block
	epochNewCh          chan *tpcmm.EpochInfo
	epochInterval       uint64 //the height number between two epochs
	currentEpoch        *tpcmm.EpochInfo
	dkgStartBeforeEpoch uint64 //the starting height number of DKG before an epoch
	exeScheduler        execution.ExecutionScheduler
	ledger              ledger.Ledger
	dkgExchange         *dkgExchange
	activeNodeInfos     *activeNodeInfos
}

func NewEpochService(log tplog.Logger,
	nodeID string,
	blockAddedCh chan *tpchaintypes.Block,
	epochNewCh chan *tpcmm.EpochInfo,
	epochInterval uint64,
	dkgStartBeforeEpoch uint64,
	exeScheduler execution.ExecutionScheduler,
	ledger ledger.Ledger,
	dkgExchange *dkgExchange) EpochService {
	if ledger == nil || dkgExchange == nil {
		log.Panic("Invalid input parameter and can't create epoch service!")
	}

	compStateRN := state.CreateCompositionStateReadonly(log, ledger)
	defer compStateRN.Stop()

	currentEpoch, err := compStateRN.GetLatestEpoch()
	if err != nil {
		log.Panicf("Can't get the latest epoch: err=%v", err)
	}

	anInfos := &activeNodeInfos{}
	anInfos.update(log, compStateRN)

	return &epochService{
		log:                 log,
		nodeID:              nodeID,
		blockAddedCh:        blockAddedCh,
		epochNewCh:          epochNewCh,
		epochInterval:       epochInterval,
		currentEpoch:        currentEpoch,
		dkgStartBeforeEpoch: dkgStartBeforeEpoch,
		exeScheduler:        exeScheduler,
		ledger:              ledger,
		dkgExchange:         dkgExchange,
		activeNodeInfos:     anInfos,
	}
}

func (an *activeNodeInfos) update(log tplog.Logger, compStateRN state.CompositionStateReadonly) {
	an.sync.Lock()
	defer an.sync.Unlock()

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
	proposerInfos, err := compStateRN.GetAllActiveProposers()
	if err != nil {
		log.Errorf("Get active proposer infos failed: %v", err)
		return
	}
	validatorInfos, err := compStateRN.GetAllActiveValidators()
	if err != nil {
		log.Errorf("Get active validator infos failed: %v", err)
		return
	}

	for _, executorInfo := range executorInfos {
		an.activeExecutorIDs = append(an.activeExecutorIDs, executorInfo.NodeID)
		an.activeExecutorWeights += executorInfo.Weight
		an.nodeWeights[executorInfo.NodeID] = executorInfo.Weight
	}

	for _, proposerInfo := range proposerInfos {
		an.activeProposerIDs = append(an.activeProposerIDs, proposerInfo.NodeID)
		an.activeProposerWeights += proposerInfo.Weight
		an.nodeWeights[proposerInfo.NodeID] = proposerInfo.Weight
	}

	for _, validatorInfo := range validatorInfos {
		an.activeValidatorIDs = append(an.activeValidatorIDs, validatorInfo.NodeID)
		an.activeValidatorWeights += validatorInfo.Weight
		an.nodeWeights[validatorInfo.NodeID] = validatorInfo.Weight
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

func (es *epochService) processBlockAddedStart(ctx context.Context) {
	go func() {
		for {
			select {
			case newBh := <-es.blockAddedCh:
				err := func() error {
					es.log.Infof("Epoch services receive new block added event: new block height %d", newBh.Head.Height)

					epochInfo := es.currentEpoch

					deltaH := int(newBh.Head.Height) - int(epochInfo.StartHeight)
					if deltaH == int(es.epochInterval)-int(es.dkgStartBeforeEpoch) {
						dkgExState := es.dkgExchange.getDKGState()
						if dkgExState != DKGExchangeState_IDLE {
							es.log.Warnf("Unfinished dkg exchange is going on an new dkg can't start, current state: %s", dkgExState.String)
						} else {
							compStateRN := state.CreateCompositionStateReadonly(es.log, es.ledger)
							defer compStateRN.Stop()

							err := es.dkgExchange.updateDKGPartPubKeys(compStateRN)
							if err != nil {
								es.log.Errorf("Update DKG exchange part pub keys err: %v", err)
								return err
							}
							es.dkgExchange.initWhenStart(epochInfo.Epoch + 1)
							es.dkgExchange.start(epochInfo.Epoch + 1)
						}
					}

					return nil
				}()

				if err != nil {
					continue
				}
			case <-ctx.Done():
				es.log.Info("Epoch service propcess block added exit")
				return
			}
		}
	}()
}

func (es *epochService) processEpochNewStart(ctx context.Context) {
	go func() {
		for {
			select {
			case newEpoch := <-es.epochNewCh:
				es.log.Infof("Epoch service receives new epoch event: new epoch %d", newEpoch.Epoch)
				es.currentEpoch = newEpoch

				compStateRN := state.CreateCompositionStateReadonly(es.log, es.ledger)
				es.activeNodeInfos.update(es.log, compStateRN)
				if !es.dkgExchange.dkgCrypt.Finished() {
					es.log.Warnf("Current epoch %d DKG unfinished and still use the old, height %d", newEpoch.Epoch, newEpoch.StartHeight)
				} else {
					es.dkgExchange.notifyUpdater()
					es.dkgExchange.updateDKGState(DKGExchangeState_IDLE)
				}
				compStateRN.Stop()
			case <-ctx.Done():
				es.log.Info("Epoch service propcess epoch new exit")
				return
			}
		}
	}()
}

func (es *epochService) Start(ctx context.Context) {
	es.processBlockAddedStart(ctx)
	es.processEpochNewStart(ctx)
}
