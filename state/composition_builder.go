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
			compStateOfNodes: make(map[string]*compositionStateOfNode),
		}
	})

	return stateBuilder
}

type compositionStateOfNode struct {
	sync          sync.RWMutex
	nodeID        string
	createdRecord map[uint64]bool
	compStates    map[uint64]CompositionState //StateVersion->CompositionState
}

type CompositionStateBuilder struct {
	sync             sync.RWMutex
	compStateOfNodes map[string]*compositionStateOfNode //nodeID->compositionStateNode
}

func (builder *CompositionStateBuilder) getCompositionStateOfNode(nodeID string) *compositionStateOfNode {
	builder.sync.Lock()
	defer builder.sync.Unlock()

	compStateOfNode, ok := builder.compStateOfNodes[nodeID]
	if !ok {
		builder.compStateOfNodes[nodeID] = &compositionStateOfNode{
			nodeID:        nodeID,
			createdRecord: make(map[uint64]bool),
			compStates:    make(map[uint64]CompositionState),
		}
		compStateOfNode = builder.compStateOfNodes[nodeID]
	}

	return compStateOfNode
}

func (builder *CompositionStateBuilder) createCompositionStateOfNode(log tplog.Logger, compStateOfNode *compositionStateOfNode, ledger ledger.Ledger, stateVersion uint64, requester string) CompositionState {
	compStateOfNode.sync.Lock()
	defer compStateOfNode.sync.Unlock()

	needCreation := true
	availCompStateCnt := 0
	var compStateRTN CompositionState
	var availCompStateVersions []uint64
	for sVer, compState := range compStateOfNode.compStates {
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

	if availCompStateCnt >= MaxAvail_Count && needCreation {
		log.Errorf("Can't create new CompositionState because of reaching max available value %d: availCompStateCnt %d, stateVersion %d, availCompStateVersions %v, requester=%s, self node %s",
			MaxAvail_Count, availCompStateCnt, stateVersion, availCompStateVersions, requester, compStateOfNode.nodeID)
		return nil
	}

	if needCreation {
		if _, ok := compStateOfNode.createdRecord[stateVersion]; !ok {
			compStateOfNode.createdRecord[stateVersion] = true
			compStateRTN = CreateCompositionState(log, ledger, stateVersion)
			compStateOfNode.compStates[stateVersion] = compStateRTN
			log.Infof("Create new CompositionState for stateVersion %d，requester=%s, self node %s", stateVersion, requester, compStateOfNode.nodeID)
		} else {
			log.Warnf("Have created CompositionState for stateVersion %d，so ignore the create request, requester=%s, self node %s", stateVersion, requester, compStateOfNode.nodeID)
		}
	}

	return compStateRTN
}

func (builder *CompositionStateBuilder) CreateCompositionState(log tplog.Logger, nodeID string, ledger ledger.Ledger, stateVersion uint64, requester string) CompositionState {
	compStateOfNode := builder.getCompositionStateOfNode(nodeID)

	return builder.createCompositionStateOfNode(log, compStateOfNode, ledger, stateVersion, requester)
}

func (builder *CompositionStateBuilder) CompositionState(nodeID string, stateVersion uint64) CompositionState {
	compStateOfNode := builder.getCompositionStateOfNode(nodeID)

	compStateOfNode.sync.RLock()
	defer compStateOfNode.sync.RUnlock()

	if compState, ok := compStateOfNode.compStates[stateVersion]; ok {
		return compState
	}

	return nil
}
