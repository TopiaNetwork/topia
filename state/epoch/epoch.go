package epoch

import (
	"encoding/json"

	"github.com/TopiaNetwork/topia/chain"
	tplgss "github.com/TopiaNetwork/topia/ledger/state"
)

const StateStore_Name_Epoch = "epoch"

const (
	LatestEpoch_Key = "latestepoch"
)

type EpochState interface {
	GetRoundStateRoot() ([]byte, error)

	GetLatestEpoch() (*chain.EpochInfo, error)

	SetLatestEpoch(epoch *chain.EpochInfo) error
}

type epochState struct {
	tplgss.StateStore
}

func NewRoundState(stateStore tplgss.StateStore) EpochState {
	stateStore.AddNamedStateStore("epoch")
	return &epochState{
		StateStore: stateStore,
	}
}

func (es *epochState) GetRoundStateRoot() ([]byte, error) {
	return es.Root(StateStore_Name_Epoch)
}

func (es *epochState) GetLatestEpoch() (*chain.EpochInfo, error) {
	latestEpochBytes, _, err := es.GetState(StateStore_Name_Epoch, []byte(LatestEpoch_Key))
	if err != nil || latestEpochBytes == nil {
		return nil, err
	}

	var eponInfo chain.EpochInfo
	err = json.Unmarshal(latestEpochBytes, &eponInfo)
	if err != nil {
		return nil, err
	}

	return &eponInfo, nil
}

func (es *epochState) SetLatestEpoch(epoch *chain.EpochInfo) error {
	epochBytes, err := json.Marshal(epoch)
	if err != nil {
		return err
	}

	isExist, _ := es.Exists(StateStore_Name_Epoch, []byte(LatestEpoch_Key))
	if isExist {
		return es.Update(StateStore_Name_Epoch, []byte(LatestEpoch_Key), epochBytes)
	} else {
		return es.Put(StateStore_Name_Epoch, []byte(LatestEpoch_Key), epochBytes)
	}
}
