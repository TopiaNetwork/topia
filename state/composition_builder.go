package state

import (
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
			compStateMap: make(map[string]map[uint64]CompositionState),
		}
	})

	return stateBuilder
}

type CompositionStateBuilder struct {
	sync         sync.RWMutex
	compStateMap map[string]map[uint64]CompositionState //nodeID->map[uint64]CompositionState(StateVersion->CompositionState)
}

func (builder *CompositionStateBuilder) CreateCompositionState(log tplog.Logger, nodeID string, ledger ledger.Ledger, stateVersion uint64) CompositionState {
	builder.sync.Lock()
	defer builder.sync.Unlock()

	compStateVerMap, ok := builder.compStateMap[nodeID]
	if !ok {
		compStateVerMap = make(map[uint64]CompositionState)
		builder.compStateMap[nodeID] = compStateVerMap
	}

	var compStateRTN CompositionState

	availCompStateCnt := 0
	for sVer, compState := range compStateVerMap {
		if compState.CompSState() == CompSState_Commited {
			delete(compStateVerMap, stateVersion)

			if sVer == stateVersion {
				log.Warnf("Existed CompositionState for stateVersion %d has been commited, so ignore subsequent disposing", stateVersion)
				delete(compStateVerMap, stateVersion)
				return nil
			}
		} else {
			availCompStateCnt++
			if sVer == stateVersion {
				log.Infof("Existed CompositionState for stateVersion %d", stateVersion)
				return compState
			}
		}

		if availCompStateCnt >= MaxAvail_Count {
			log.Errorf("Can't create new CompositionState because of reaching max available value %d: stateVersion %d", MaxAvail_Count, stateVersion)
			return nil
		}
	}

	if availCompStateCnt >= MaxAvail_Count {
		log.Errorf("Can't create new CompositionState because of reaching max available value %d: stateVersion %d", MaxAvail_Count, stateVersion)
		return nil
	}

	if compStateRTN == nil {
		compStateRTN = CreateCompositionState(log, ledger)
		compStateVerMap[stateVersion] = compStateRTN
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
