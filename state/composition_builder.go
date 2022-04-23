package state

import (
	"fmt"
	"github.com/TopiaNetwork/topia/ledger"
	tplog "github.com/TopiaNetwork/topia/log"
	"sync"
)

/* CompositionStateBuilder is only for proposer and validator
   and new CompositionState will be created nonce received new added block to the chain.
   For executor, a new CompositionState will be created when a prepare packed txs created.
*/

var stateBuilder *CompositionStateBuilder
var once sync.Once

const Wait_StateStore_Time = 50 //ms
const MaxAvail_Count = 3

func GetStateBuilder() *CompositionStateBuilder {
	once.Do(func() {
		stateBuilder = &CompositionStateBuilder{
			createdRecord: make(map[string]map[uint64]bool),
			compStateMap:  make(map[string]map[uint64]CompositionState),
		}
	})

	return stateBuilder
}

type CompositionStateBuilder struct {
	sync          sync.RWMutex
	createdRecord map[string]map[uint64]bool
	compStateMap  map[string]map[uint64]CompositionState //nodeID->map[uint64]CompositionState(StateVersion->CompositionState)
}

func (builder *CompositionStateBuilder) CreateCompositionState(log tplog.Logger, nodeID string, ledger ledger.Ledger, stateVersion uint64, requester string) CompositionState {
	builder.sync.Lock()
	defer builder.sync.Unlock()

	compStateVerMap, ok := builder.compStateMap[nodeID]
	if !ok {
		builder.createdRecord[nodeID] = make(map[uint64]bool)
		builder.compStateMap[nodeID] = make(map[uint64]CompositionState)
		compStateVerMap = builder.compStateMap[nodeID]
	}

	needCreation := true
	availCompStateCnt := 0
	var compStateRTN CompositionState
	var availCompStateVersions []uint64
	for sVer, compState := range compStateVerMap {
		func() {
			compState.Lock()
			defer compState.Unlock()

			if compState.CompSState() == CompSState_Commited {
				if sVer == 3 {
					csStateRN := CreateCompositionStateReadonly(log, ledger)
					latestBlock, err := csStateRN.GetLatestBlock()
					if err != nil {
						err = fmt.Errorf("Can't get the latest block: %v", err)
						csStateRN.Stop()
					}
					csStateRN.Stop()

					log.Infof("latest block %d", latestBlock.Head.Height)
				}

				delete(compStateVerMap, compState.StateVersion())
				log.Infof("Delete CompositionState %d: input stateVersion %d, requester=%s, self node %s", compState.StateVersion(), stateVersion, requester, nodeID)

				if sVer == stateVersion {
					log.Warnf("Existed CompositionState for stateVersion %d has been commited, so ignore subsequent disposing, requester=%s, self node %s", stateVersion, requester, nodeID)
					compStateRTN = nil
					needCreation = false
				}
			} else {
				availCompStateCnt++
				availCompStateVersions = append(availCompStateVersions, sVer)
				if sVer == stateVersion {
					log.Infof("Existed CompositionState for stateVersion %d, requester=%s, self node %s", stateVersion, requester, nodeID)
					compStateRTN = compState
					needCreation = false
				}
			}
		}()
	}

	if availCompStateCnt >= MaxAvail_Count && needCreation {
		log.Errorf("Can't create new CompositionState because of reaching max available value %d: availCompStateCnt %d, stateVersion %d, availCompStateVersions %v, requester=%s, self node %s",
			MaxAvail_Count, availCompStateCnt, stateVersion, availCompStateVersions, requester, nodeID)
		return nil
	}

	if needCreation {
		if _, ok := builder.createdRecord[nodeID][stateVersion]; !ok {
			builder.createdRecord[nodeID][stateVersion] = true
			compStateRTN = CreateCompositionState(log, ledger, stateVersion)
			compStateVerMap[stateVersion] = compStateRTN
			log.Infof("Create new CompositionState for stateVersion %d，requester=%s, self node %s", stateVersion, requester, nodeID)
		} else {
			log.Warnf("Have created CompositionState for stateVersion %d，so ignore the create request, requester=%s, self node %s", stateVersion, requester, nodeID)
		}
	}

	return compStateRTN
}

func (builder *CompositionStateBuilder) CompositionState(nodeID string, stateVersion uint64) CompositionState {
	builder.sync.RLock()
	defer builder.sync.RUnlock()

	compStateVerMap, ok := builder.compStateMap[nodeID]
	if !ok {
		return nil
	}

	if compState, ok := compStateVerMap[stateVersion]; ok {
		return compState
	}

	return nil
}
