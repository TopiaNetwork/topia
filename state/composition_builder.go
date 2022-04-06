package state

import (
	"sync"
	"time"

	"github.com/TopiaNetwork/topia/ledger"
	tplog "github.com/TopiaNetwork/topia/log"
)

/* CompositionStateBuilder is only for proposer and validator
   and new CompositionState will be created nonce received new added block to the chain.
   For executor, a new CompositionState will be created when a prepare packed txs created.
*/

var stateBuilder *CompositionStateBuilder
var once sync.Once

const Wait_StateStore_Time = 50 //ms

func GetStateBuilder() *CompositionStateBuilder {
	once.Do(func() {
		stateBuilder = &CompositionStateBuilder{}
	})

	return stateBuilder
}

type CompositionStateBuilder struct {
	sync         sync.RWMutex
	compStateMap map[uint64]CompositionState //StateVersion->CompositionState
}

func (builder *CompositionStateBuilder) CreateCompositionState(log tplog.Logger, ledger ledger.Ledger, stateVersion uint64) CompositionState {
	builder.sync.Lock()
	defer builder.sync.Unlock()

	if compState, ok := builder.compStateMap[stateVersion]; ok {
		log.Infof("Exist CompositionState for stateVersion %d", stateVersion)
		return compState
	}

	if stateVersion > 1 {
		compState, ok := builder.compStateMap[stateVersion-1]
		if ok {
			i := 1
			for ; compState.PendingStateStore() > 0 && i <= 3; i++ {
				log.Warnf("Last CompositionState hasn't been commited, need waiting for %d ms, no. %d ", Wait_StateStore_Time, i)
				time.Sleep(Wait_StateStore_Time * time.Millisecond)
			}
			if i > 3 {
				log.Errorf("Can't create new CompositionState because of last state version not been commited: stateVersion %d", stateVersion)
				return nil
			}

			delete(builder.compStateMap, stateVersion-1)
		}
	}

	compState := CreateCompositionState(log, ledger)

	builder.compStateMap[stateVersion] = compState

	return compState
}

func (builder *CompositionStateBuilder) CompositionState(stateVersion uint64) CompositionState {
	builder.sync.RLock()
	defer builder.sync.RUnlock()

	if compState, ok := builder.compStateMap[stateVersion]; ok {
		return compState
	}

	return nil
}
