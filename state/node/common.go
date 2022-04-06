package node

import (
	"encoding/binary"
	"encoding/json"
	"errors"

	"github.com/TopiaNetwork/topia/chain"
	tpcmm "github.com/TopiaNetwork/topia/common"
	tplgss "github.com/TopiaNetwork/topia/ledger/state"
)

func isNodeExist(stateStore tplgss.StateStore, name string, nodeID string) bool {
	isExist, _ := stateStore.Exists(name, []byte(nodeID))

	return isExist
}

func getNode(stateStore tplgss.StateStore, name string, nodeID string) (*chain.NodeInfo, error) {
	nodeInfoBytes, _, err := stateStore.GetState(name, []byte(nodeID))
	if err != nil {
		return nil, err
	}

	if nodeInfoBytes == nil {
		return nil, nil
	}

	var nodeInfo chain.NodeInfo
	err = json.Unmarshal(nodeInfoBytes, &nodeInfo)
	if err != nil {
		return nil, err
	}

	return &nodeInfo, nil
}

func addNode(stateStore tplgss.StateStore, name string, totalNodeIDsKey string, toalWeightKey string, nodeInfo *chain.NodeInfo) error {
	if nodeInfo == nil {
		return errors.New("Nil node info")
	}

	nodeInfoBytes, err := json.Marshal(nodeInfo)
	if err != nil {
		return err
	}

	err = stateStore.Put(name, []byte(nodeInfo.NodeID), nodeInfoBytes)
	if err != nil {
		return err
	}

	var errT error
	isTotalIDKey, err := stateStore.Exists(name, []byte(totalNodeIDsKey))
	if isTotalIDKey {
		errT = updateTotalIDs(stateStore, name, totalNodeIDsKey, nodeInfo.NodeID, false)
	} else {
		totolIdsBytes, err := json.Marshal(&[]string{nodeInfo.NodeID})
		if err != nil {
			return err
		}

		errT = stateStore.Put(name, []byte(totalNodeIDsKey), totolIdsBytes)
	}
	if errT != nil {
		return errT
	}

	isExist, err := stateStore.Exists(name, []byte(toalWeightKey))
	if isExist {
		return updateTotalWeight(stateStore, name, toalWeightKey, int64(nodeInfo.Weight))
	} else {
		totalWeightBytes := make([]byte, 8)
		binary.BigEndian.PutUint64(totalWeightBytes, nodeInfo.Weight)
		return stateStore.Put(name, []byte(toalWeightKey), totalWeightBytes)
	}
}

func updateTotalIDs(stateStore tplgss.StateStore, name string, totalIDsKey string, nodeID string, isRemove bool) error {
	totolIdsBytes, _, err := stateStore.GetState(name, []byte(totalIDsKey))
	if err != nil {
		return err
	}

	var nodeIDs []string
	err = json.Unmarshal(totolIdsBytes, &nodeIDs)
	if err != nil {
		return err
	}

	if !isRemove {
		nodeIDs = append(nodeIDs, nodeID)
	} else {
		nodeIDs = tpcmm.RemoveIfExistString(nodeID, nodeIDs)
	}
	finalTotolIdsBytesNew, err := json.Marshal(&nodeIDs)
	if err != nil {
		return err
	}

	return stateStore.Update(StateStore_Name_Node, []byte(totalIDsKey), finalTotolIdsBytesNew)
}

func updateTotalWeight(stateStore tplgss.StateStore, name string, totalWeightKey string, deltaWeight int64) error {
	totalWeightBytes, _, err := stateStore.GetState(name, []byte(totalWeightKey))
	if err != nil {
		return err
	}

	curTotalWeight := binary.BigEndian.Uint64(totalWeightBytes)

	finalTotalWeight := int64(curTotalWeight) + deltaWeight
	if finalTotalWeight < 0 {
		finalTotalWeight = 0
	}

	finalTotalWeightBytes := make([]byte, 8)
	binary.BigEndian.PutUint64(finalTotalWeightBytes, uint64(finalTotalWeight))

	return stateStore.Update(name, []byte(totalWeightKey), finalTotalWeightBytes)
}

func uppdateWeight(stateStore tplgss.StateStore, name string, nodeID string, totalWeightKey string, weight uint64) error {
	nodeInfo, err := getNode(stateStore, name, nodeID)
	if err != nil {
		return err
	}

	deltaWeight := int64(weight) - int64(nodeInfo.Weight)

	nodeInfo.Weight = weight

	nodeInfoBytes, err := json.Marshal(nodeInfo)
	if err != nil {
		return err
	}

	err = stateStore.Put(name, []byte(nodeInfo.NodeID), nodeInfoBytes)
	if err != nil {
		return err
	}

	return updateTotalWeight(stateStore, name, totalWeightKey, deltaWeight)
}

func removeNode(stateStore tplgss.StateStore, name string, totalNodeIDsKey string, toalWeightKey string, nodeID string, wieght uint64) error {
	err := stateStore.Delete(name, []byte(nodeID))
	if err != nil {
		return err
	}

	var errT error
	isTotalIDKey, _ := stateStore.Exists(name, []byte(totalNodeIDsKey))
	if isTotalIDKey {
		errT = updateTotalIDs(stateStore, name, totalNodeIDsKey, nodeID, true)
	}
	if errT != nil {
		return errT
	}

	isExist, err := stateStore.Exists(name, []byte(toalWeightKey))
	if isExist {
		return updateTotalWeight(stateStore, name, toalWeightKey, int64(-wieght))
	}

	return nil
}
