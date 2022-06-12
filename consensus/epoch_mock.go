package consensus

import (
	"context"
	"time"

	tpcmm "github.com/TopiaNetwork/topia/common"
)

type EpochServiceMock struct{}

func (m *EpochServiceMock) GetActiveExecutorIDs() []string {
	return []string{
		"16Uiu2HAmUnckEUPdYJ35h6DkUbNKDchbBDUd6heZi5DgBHkBJRCT",
		"16Uiu2HAmKcwN8p9PZg4mnTSyruJpUNP4Crr9gfcrqPovLQjwYNhi",
		"16Uiu2HAmA4RpFMvce7JAMUgbfRGcLW4EqaYzHYefRNrSQauBwNUm",
	}
}

func (m *EpochServiceMock) GetActiveProposerIDs() []string {
	//TODO implement me
	panic("implement me")
}

func (m *EpochServiceMock) GetActiveValidatorIDs() []string {
	//TODO implement me
	panic("implement me")

}

func (m *EpochServiceMock) GetNodeWeight(nodeID string) (uint64, error) {
	nwMap :=
		map[string]uint64{
			"16Uiu2HAmUnckEUPdYJ35h6DkUbNKDchbBDUd6heZi5DgBHkBJRCT": 10,
			"16Uiu2HAmKcwN8p9PZg4mnTSyruJpUNP4Crr9gfcrqPovLQjwYNhi": 20,
			"16Uiu2HAmA4RpFMvce7JAMUgbfRGcLW4EqaYzHYefRNrSQauBwNUm": 10,
		}

	return nwMap[nodeID], nil
}

func (m *EpochServiceMock) GetActiveExecutorsTotalWeight() uint64 {
	return 30
}

func (m *EpochServiceMock) GetActiveProposersTotalWeight() uint64 {
	//TODO implement me
	panic("implement me")
}

func (m *EpochServiceMock) GetActiveValidatorsTotalWeight() uint64 {
	//TODO implement me
	panic("implement me")
}

func (m *EpochServiceMock) GetLatestEpoch() *tpcmm.EpochInfo {
	return &tpcmm.EpochInfo{
		Epoch:          0,
		StartTimeStamp: uint64(time.Now().UnixNano()),
		StartHeight:    1,
	}
}

func (m *EpochServiceMock) Start(ctx context.Context) {
	//TODO implement me
	panic("implement me")
}