package round

import tplgss "github.com/TopiaNetwork/topia/ledger/state"

type RoundState interface {
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

func (rs *roundState) GetCurrentRound() uint64 {
	//TODO implement me
	panic("implement me")
}

func (rs *roundState) SetCurrentRound(round uint64) {
	//TODO implement me
	panic("implement me")
}

func (rs *roundState) GetCurrentEpoch() uint64 {
	//TODO implement me
	panic("implement me")
}

func (rs *roundState) SetCurrentEpoch(epoch uint64) {
	//TODO implement me
	panic("implement me")
}
