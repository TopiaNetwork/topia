package round

import (
	"encoding/binary"
	tplgss "github.com/TopiaNetwork/topia/ledger/state"
)

const StateStore_Name_Round = "round"

const (
	LatestEpoch_Key = "latestepoch"
	LatestRound_Key = "latestround"
)

type RoundState interface {
	GetRoundStateRoot() ([]byte, error)

	GetCurrentRound() uint64

	SetCurrentRound(round uint64)

	GetCurrentEpoch() uint64

	SetCurrentEpoch(epoch uint64)
}

type roundState struct {
	tplgss.StateStore
}

func NewRoundState(stateStore tplgss.StateStore) RoundState {
	stateStore.AddNamedStateStore("round")
	return &roundState{
		StateStore: stateStore,
	}
}

func (rs *roundState) GetRoundStateRoot() ([]byte, error) {
	return rs.Root(StateStore_Name_Round)
}

func (rs *roundState) GetCurrentRound() uint64 {
	latestRoundBytes, _, err := rs.GetState(StateStore_Name_Round, []byte(LatestRound_Key))
	if err != nil || latestRoundBytes == nil {
		return 0
	}

	return binary.BigEndian.Uint64(latestRoundBytes)
}

func (rs *roundState) SetCurrentRound(round uint64) {
	roundBytes := make([]byte, 8)
	binary.BigEndian.PutUint64(roundBytes, round)

	isExist, _ := rs.Exists(StateStore_Name_Round, []byte(LatestRound_Key))
	if isExist {
		rs.Update(StateStore_Name_Round, []byte(LatestRound_Key), roundBytes)
	} else {
		rs.Put(StateStore_Name_Round, []byte(LatestRound_Key), roundBytes)
	}
}

func (rs *roundState) GetCurrentEpoch() uint64 {
	latestEpochBytes, _, err := rs.GetState(StateStore_Name_Round, []byte(LatestEpoch_Key))
	if err != nil || latestEpochBytes == nil {
		return 0
	}

	return binary.BigEndian.Uint64(latestEpochBytes)
}

func (rs *roundState) SetCurrentEpoch(epoch uint64) {
	epochBytes := make([]byte, 8)
	binary.BigEndian.PutUint64(epochBytes, epoch)

	isExist, _ := rs.Exists(StateStore_Name_Round, []byte(LatestEpoch_Key))
	if isExist {
		rs.Update(StateStore_Name_Round, []byte(LatestEpoch_Key), epochBytes)
	} else {
		rs.Put(StateStore_Name_Round, []byte(LatestEpoch_Key), epochBytes)
	}
}
