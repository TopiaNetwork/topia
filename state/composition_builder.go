package state

import (
	"context"
	"strconv"
	"sync"
	"sync/atomic"
	"time"

	"github.com/orcaman/concurrent-map"

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
	sync          sync.RWMutex
	nodeID        string
	createdRecord map[uint64]bool
	compStates    map[uint64]CompositionState //StateVersion->CompositionState
}

type compositionStateOfNodeSimple struct {
	nodeID          string
	isCreating      uint32
	maxStateVersion uint64
	compStates      cmap.ConcurrentMap //StateVersion->CompositionState
}

type CompositionStateBuilder interface {
	CreateCompositionState(log tplog.Logger, nodeID string, ledger ledger.Ledger, stateVersion uint64, requester string) CompositionState
	CompositionState(nodeID string, stateVersion uint64) CompositionState
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
			createdRecord: make(map[uint64]bool),
			compStates:    make(map[uint64]CompositionState),
		}

		bf.compStateOfNodes.Store(nodeID, compStateOfNode)
	}

	return compStateOfNode
}

func (bf *compositionStateBuilderFull) createCompositionStateOfNode(log tplog.Logger, compStateOfNode *compositionStateOfNodeFull, ledger ledger.Ledger, stateVersion uint64, requester string) CompositionState {
	compStateOfNode.sync.Lock()
	defer compStateOfNode.sync.Unlock()

	needCreation := true
	availCompStateCnt := 0
	var compStateRTN CompositionState
	var availCompStateVersions []uint64
	for sVer, compState := range compStateOfNode.compStates {
		func() {
			//compState.Lock()
			//defer compState.Unlock()

			log.Infof("CompositionState %d: input stateVersion %d, requester=%s, sVer=%d, self node %s", compState.StateVersion(), stateVersion, requester, sVer, compStateOfNode.nodeID)

			if compState.CompSState() == CompSState_Commited {
				delete(compStateOfNode.compStates, compState.StateVersion())
				log.Infof("Delete CompositionState %d: input stateVersion %d, requester=%s, self node %s", compState.StateVersion(), stateVersion, requester, compStateOfNode.nodeID)

				if sVer == stateVersion {
					log.Warnf("Existed CompositionState for stateVersion %d has been commited, so ignore subsequent disposing, requester=%s, self node %s", stateVersion, requester, compStateOfNode.nodeID)
					compStateRTN = nil
					needCreation = false
				}
			} else {
				availCompStateCnt++
				availCompStateVersions = append(availCompStateVersions, sVer)
				if sVer == stateVersion {
					log.Infof("Existed CompositionState for stateVersion %d, requester=%s, self node %s", stateVersion, requester, compStateOfNode.nodeID)
					compStateRTN = compState
					needCreation = false
				}
			}
		}()
	}

	log.Infof("Avail CompositionState: input stateVersion %d, requester=%s, availCompStateCnt=%d, needCreation=%v, self node %s", stateVersion, requester, availCompStateCnt, needCreation, compStateOfNode.nodeID)
	if availCompStateCnt >= MaxAvail_Count && needCreation {
		log.Errorf("Can't create new CompositionState because of reaching max available value %d: availCompStateCnt %d, stateVersion %d, availCompStateVersions %v, requester=%s, self node %s",
			MaxAvail_Count, availCompStateCnt, stateVersion, availCompStateVersions, requester, compStateOfNode.nodeID)
		return nil
	}

	if needCreation {
		if _, ok := compStateOfNode.createdRecord[stateVersion]; !ok {
			compStateOfNode.createdRecord[stateVersion] = true
			compStateRTN = createCompositionState(log, ledger, stateVersion)
			compStateOfNode.compStates[stateVersion] = compStateRTN
			log.Infof("Create new CompositionState for stateVersion %d，requester=%s, self node %s", stateVersion, requester, compStateOfNode.nodeID)
		} else {
			log.Warnf("Have created CompositionState for stateVersion %d，so ignore the create request, requester=%s, self node %s", stateVersion, requester, compStateOfNode.nodeID)
		}
	}

	return compStateRTN
}

func (bf *compositionStateBuilderFull) CreateCompositionState(log tplog.Logger, nodeID string, ledger ledger.Ledger, stateVersion uint64, requester string) CompositionState {
	compStateOfNode := bf.getCompositionStateOfNode(nodeID)

	return bf.createCompositionStateOfNode(log, compStateOfNode, ledger, stateVersion, requester)
}

func (bf *compositionStateBuilderFull) CompositionState(nodeID string, stateVersion uint64) CompositionState {
	compStateOfNode := bf.getCompositionStateOfNode(nodeID)

	compStateOfNode.sync.RLock()
	defer compStateOfNode.sync.RUnlock()

	if compState, ok := compStateOfNode.compStates[stateVersion]; ok {
		return compState
	}

	return nil
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
						for !atomic.CompareAndSwapUint32(&ns.isCreating, 0, 1) {
							strInfo := "New compositionState is creating and maintainTimer will wait: " + ns.nodeID
							log.Infof("%s", strInfo)
							time.Sleep(100 * time.Millisecond)
						}
						defer func() {
							ns.isCreating = 0
						}()

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
		if compState, can := val.(CompositionState); can {
			if can {
				compStateRTN = compState
			}

			if compState.CompSState() == CompSState_Commited {
				compStateOfNode.compStates.Remove(stateVerS)
				compStateRTN = nil
				log.Infof("Delete CompositionState when found %d: input stateVersion %d, requester=%s, self node %s", compState.StateVersion(), stateVersion, requester, compStateOfNode.nodeID)
			}
		}
	} else {
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
			compStateRTN = createCompositionState(log, ledger, stateVersion)
			compStateOfNode.compStates.Set(stateVerS, compStateRTN)

			compStateOfNode.maxStateVersion = stateVersion

			log.Infof("Create new CompositionState for stateVersion %d，requester=%s, self node %s", stateVersion, requester, compStateOfNode.nodeID)
		}
	}

	return compStateRTN
}

func (bs *compositionStateBuilderSimple) CreateCompositionState(log tplog.Logger, nodeID string, ledger ledger.Ledger, stateVersion uint64, requester string) CompositionState {
	compStateOfNode := bs.getCompositionStateOfNode(nodeID, log, ledger)

	return bs.createCompositionStateOfNode(log, compStateOfNode, ledger, stateVersion, requester)
}

func (bs *compositionStateBuilderSimple) CompositionState(nodeID string, stateVersion uint64) CompositionState {
	compStateOfNode := bs.getCompositionStateOfNode(nodeID, nil, nil)
	if compStateOfNode == nil {
		return nil
	}

	stateVerS := strconv.FormatUint(stateVersion, 10)
	if val, ok := compStateOfNode.compStates.Get(stateVerS); ok {
		if compState, can := val.(CompositionState); can {
			if can {
				return compState
			}
		}
	}

	return nil
}
