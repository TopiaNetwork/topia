package epoch

import (
	"encoding/json"

	tpcmm "github.com/TopiaNetwork/topia/common"
	tplgss "github.com/TopiaNetwork/topia/ledger/state"
)

const StateStore_Name_Epoch = "epoch"

const (
	LatestEpoch_Key = "latestepoch"
)

type EpochState interface {
	GetEpochRoot() ([]byte, error)

	GetLatestEpoch() (*tpcmm.EpochInfo, error)

	SetLatestEpoch(epoch *tpcmm.EpochInfo) error
}

type epochState struct {
	tplgss.StateStore
}

func NewRoundState(stateStore tplgss.StateStore, cacheSize int) EpochState {
	stateStore.AddNamedStateStore(StateStore_Name_Epoch, cacheSize)
	return &epochState{
		StateStore: stateStore,
	}
}

func (es *epochState) GetEpochRoot() ([]byte, error) {
	return es.Root(StateStore_Name_Epoch)
}

func (es *epochState) GetLatestEpoch() (*tpcmm.EpochInfo, error) {
	latestEpochBytes, err := es.GetStateData(StateStore_Name_Epoch, []byte(LatestEpoch_Key))
	if err != nil || latestEpochBytes == nil {
		return nil, err
	}

	var eponInfo tpcmm.EpochInfo
	err = json.Unmarshal(latestEpochBytes, &eponInfo)
	if err != nil {
		return nil, err
	}

	return &eponInfo, nil
}

func (es *epochState) SetLatestEpoch(epoch *tpcmm.EpochInfo) error {
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
