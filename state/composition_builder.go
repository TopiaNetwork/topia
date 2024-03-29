package state

import (
	"context"
	"github.com/orcaman/concurrent-map"
	"strconv"
	"sync"
	"time"

	"github.com/TopiaNetwork/topia/ledger"
	tplog "github.com/TopiaNetwork/topia/log"
)

/* CompositionStateBuilder is only for proposer and validator
   and new CompositionState will be created nonce received new added block to the chain.
   For executor, a new CompositionState will be created when a prepare packed txs created.
*/

var stateBuilderFull, stateBuilderSimple CompositionStateBuilder
var onceFull, onceSimple sync.Once

const Wait_StateStore_Time = 50 //ms
const MaxAvail_Count = 5

type CompStateBuilderType byte

const (
	CompStateBuilderType_Unknown CompStateBuilderType = iota
	CompStateBuilderType_Full
	CompStateBuilderType_Simple
)

func getStateBuilderFull() CompositionStateBuilder {
	onceFull.Do(func() {
		stateBuilderFull = &compositionStateBuilderFull{}
	})

	return stateBuilderFull
}

func getStateBuilderSimple() CompositionStateBuilder {
	onceSimple.Do(func() {
		stateBuilderSimple = &compositionStateBuilderSimple{}
	})

	return stateBuilderSimple
}

func GetStateBuilder(cType CompStateBuilderType) CompositionStateBuilder {
	switch cType {
	case CompStateBuilderType_Full:
		return getStateBuilderFull()
	case CompStateBuilderType_Simple:
		return getStateBuilderSimple()
	default:
		panic("Unknown cType")
	}

	return nil
}

type compositionStateOfNodeFull struct {
	sync                 sync.RWMutex
	nodeID               string
	lastCompositionState CompositionState
	createdRecord        map[uint64]struct{}
}

type compositionStateOfNodeSimple struct {
	nodeID          string
	isCreating      uint32
	maxStateVersion uint64
	compStates      cmap.ConcurrentMap //StateVersion->CompositionState
}

type CompositionStateBuilder interface {
	CreateCompositionState(log tplog.Logger, nodeID string, ledger ledger.Ledger, stateVersion uint64, requester string) CompositionState

	CompositionStateExist(nodeID string, stateVersion uint64) bool

	CompositionStateAtVersion(nodeID string, stateVersion uint64) CompositionState
}

type compositionStateBuilderFull struct {
	compStateOfNodes sync.Map //nodeID->compositionStateNodeFull
}

type compositionStateBuilderSimple struct {
	compStateOfNodes sync.Map //nodeID->compositionStateNodeSimple
}

func (bf *compositionStateBuilderFull) getCompositionStateOfNode(nodeID string) *compositionStateOfNodeFull {
	var compStateOfNode *compositionStateOfNodeFull
	if val, ok := bf.compStateOfNodes.Load(nodeID); ok {
		compStateOfNode = val.(*compositionStateOfNodeFull)
	} else {
		compStateOfNode = &compositionStateOfNodeFull{
			nodeID:        nodeID,
			createdRecord: make(map[uint64]struct{}),
		}

		bf.compStateOfNodes.Store(nodeID, compStateOfNode)
	}

	return compStateOfNode
}

func (bf *compositionStateBuilderFull) createCompositionStateOfNode(log tplog.Logger, compStateOfNode *compositionStateOfNodeFull, ledger ledger.Ledger, stateVersion uint64, requester string) CompositionState {
	compStateOfNode.sync.Lock()
	defer compStateOfNode.sync.Unlock()

	var compStateRTN CompositionState
	if _, ok := compStateOfNode.createdRecord[stateVersion]; !ok {
		compStateOfNode.createdRecord[stateVersion] = struct{}{}
		compStateRTN = createCompositionState(log, ledger, stateVersion)
		compStateOfNode.lastCompositionState = compStateRTN
		log.Infof("Create new CompositionState for stateVersion %d，requester=%s, self node %s", stateVersion, requester, compStateOfNode.nodeID)
	} else {
		log.Warnf("Have created CompositionState for stateVersion %d，so use the last, requester=%s, self node %s", stateVersion, requester, compStateOfNode.nodeID)
		if compStateOfNode.lastCompositionState != nil &&
			compStateOfNode.lastCompositionState.StateVersion() == stateVersion &&
			compStateOfNode.lastCompositionState.CompSState() != CompSState_Commited {
			compStateRTN = compStateOfNode.lastCompositionState
		}
	}

	return compStateRTN
}

func (bf *compositionStateBuilderFull) CreateCompositionState(log tplog.Logger, nodeID string, ledger ledger.Ledger, stateVersion uint64, requester string) CompositionState {
	compStateOfNode := bf.getCompositionStateOfNode(nodeID)

	return bf.createCompositionStateOfNode(log, compStateOfNode, ledger, stateVersion, requester)
}

func (bf *compositionStateBuilderFull) CompositionStateExist(nodeID string, stateVersion uint64) bool {
	compStateOfNode := bf.getCompositionStateOfNode(nodeID)

	compStateOfNode.sync.RLock()
	defer compStateOfNode.sync.RUnlock()

	if _, ok := compStateOfNode.createdRecord[stateVersion]; ok {
		return true
	}

	return false
}

func (bs *compositionStateBuilderFull) CompositionStateAtVersion(nodeID string, stateVersion uint64) CompositionState {
	panic("Composition state builder full not support CompositionStateAtVersion")
}

func (ns *compositionStateOfNodeSimple) maintainTimerStart(ctx context.Context, log tplog.Logger, ledger ledger.Ledger) {
	go func() {
		timer := time.NewTicker(1500 * time.Millisecond)
		defer timer.Stop()
		for {
			select {
			case <-timer.C:
				var willRemoveKey []string
				ns.compStates.IterCb(func(key string, v interface{}) {
					compState := v.(CompositionState)
					if compState.CompSState() == CompSState_Commited {
						willRemoveKey = append(willRemoveKey, key)
					}
				})
				for i, _ := range willRemoveKey {
					ns.compStates.Remove(willRemoveKey[i])
				}

				if ns.compStates.Count() <= 0.6*100 {
					func() {
						/*
							for !atomic.CompareAndSwapUint32(&ns.isCreating, 0, 1) {
								strInfo := "New compositionState is creating and maintainTimer will wait: " + ns.nodeID
								log.Infof("%s", strInfo)
								time.Sleep(100 * time.Millisecond)
							}
							defer func() {
								ns.isCreating = 0
							}()
						*/

						maxStateVer := ns.maxStateVersion
						for stateVer := maxStateVer + 1; stateVer <= maxStateVer+40; stateVer++ {
							stateVerS := strconv.FormatUint(stateVer, 10)
							compStateNew := createCompositionState(log, ledger, stateVer)
							ns.compStates.Set(stateVerS, compStateNew)
						}
						ns.maxStateVersion = maxStateVer + 40
					}()
				}
			case <-ctx.Done():
				log.Infof("Maintain timer of compositionStateOfNodeSimple exit")
			}
		}
	}()

}

func (bs *compositionStateBuilderSimple) getCompositionStateOfNode(nodeID string, log tplog.Logger, ledger ledger.Ledger) *compositionStateOfNodeSimple {
	var compStateOfNode *compositionStateOfNodeSimple
	if val, ok := bs.compStateOfNodes.Load(nodeID); ok {
		compStateOfNode = val.(*compositionStateOfNodeSimple)
	} else {
		if ledger == nil {
			return nil
		}

		compStateOfNode = &compositionStateOfNodeSimple{
			nodeID:     nodeID,
			compStates: cmap.New(),
		}
		bs.compStateOfNodes.Store(nodeID, compStateOfNode)

		compStateOfNode.maintainTimerStart(context.Background(), log, ledger)
	}

	return compStateOfNode
}

func (bs *compositionStateBuilderSimple) createCompositionStateOfNode(log tplog.Logger, compStateOfNode *compositionStateOfNodeSimple, ledger ledger.Ledger, stateVersion uint64, requester string) CompositionState {
	var compStateRTN CompositionState

	stateVerS := strconv.FormatUint(stateVersion, 10)
	if val, ok := compStateOfNode.compStates.Get(stateVerS); ok {
		compStateRTN = val.(CompositionState)
		if compStateRTN.CompSState() == CompSState_Commited {
			compStateOfNode.compStates.Remove(stateVerS)
			compStateRTN = nil
			log.Infof("Delete CompositionState when found: stateVersion %d, requester=%s, self node %s", stateVersion, requester, compStateOfNode.nodeID)
		}
	} else {
		/*
			for !atomic.CompareAndSwapUint32(&compStateOfNode.isCreating, 0, 1) {
				strInfo := "New compositionState is creating and createCompositionStateOfNode will wait: " + compStateOfNode.nodeID
				log.Infof("%s", strInfo)
				time.Sleep(50 * time.Millisecond)
			}
			defer func() {
				compStateOfNode.isCreating = 0
			}()

			if val, ok := compStateOfNode.compStates.Get(stateVerS); ok {
				return val.(CompositionState)
			} else {

		*/
		compStateRTN = createCompositionState(log, ledger, stateVersion)
		compStateOfNode.compStates.Set(stateVerS, compStateRTN)

		compStateOfNode.maxStateVersion = stateVersion

		log.Infof("Create new CompositionState for stateVersion %d，requester=%s, self node %s", stateVersion, requester, compStateOfNode.nodeID)
		//}
	}

	return compStateRTN
}

func (bs *compositionStateBuilderSimple) CreateCompositionState(log tplog.Logger, nodeID string, ledger ledger.Ledger, stateVersion uint64, requester string) CompositionState {
	compStateOfNode := bs.getCompositionStateOfNode(nodeID, log, ledger)

	return bs.createCompositionStateOfNode(log, compStateOfNode, ledger, stateVersion, requester)
}

func (bs *compositionStateBuilderSimple) CompositionStateExist(nodeID string, stateVersion uint64) bool {
	compStateOfNode := bs.getCompositionStateOfNode(nodeID, nil, nil)
	if compStateOfNode == nil {
		return false
	}

	stateVerS := strconv.FormatUint(stateVersion, 10)
	if val, ok := compStateOfNode.compStates.Get(stateVerS); ok {
		if _, can := val.(CompositionState); can {
			if can {
				return true
			}
		}
	}

	return false
}

func (bs *compositionStateBuilderSimple) CompositionStateAtVersion(nodeID string, stateVersion uint64) CompositionState {
	compStateOfNode := bs.getCompositionStateOfNode(nodeID, nil, nil)
	if compStateOfNode == nil {
		return nil
	}

	stateVerS := strconv.FormatUint(stateVersion, 10)
	if topCSVal, ok := compStateOfNode.compStates.Get(stateVerS); ok {
		return topCSVal.(CompositionState)
	}

	return nil
}
