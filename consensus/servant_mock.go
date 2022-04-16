package consensus

import (
	tpchaintypes "github.com/TopiaNetwork/topia/chain/types"
	"github.com/TopiaNetwork/topia/common"
	"time"
)

type consensusServanMock struct{}

func (cs *consensusServanMock) ChainID() tpchaintypes.ChainID {
	return "testtopia"
}

func (cs *consensusServanMock) GetLatestEpoch() (*common.EpochInfo, error) {
	return &common.EpochInfo{
		Epoch:          0,
		StartTimeStamp: uint64(time.Now().UnixNano()),
		StartHeight:    1,
	}, nil
}

func (cs *consensusServanMock) GetLatestBlock() (*tpchaintypes.Block, error) {
	timeStamp := uint64(time.Now().UnixNano())

	return &tpchaintypes.Block{
		Head: &tpchaintypes.BlockHead{
			ChainID:   []byte(cs.ChainID()),
			Version:   1,
			Height:    1,
			Epoch:     0,
			Round:     1,
			TimeStamp: timeStamp,
		},
	}, nil
}

func (cs *consensusServanMock) GetAllConsensusNodeIDs() ([]string, error) {
	//TODO implement me
	panic("implement me")
}

func (cs *consensusServanMock) GetActiveExecutorIDs() ([]string, error) {
	return []string{
		"16Uiu2HAmUnckEUPdYJ35h6DkUbNKDchbBDUd6heZi5DgBHkBJRCT",
		"16Uiu2HAmKcwN8p9PZg4mnTSyruJpUNP4Crr9gfcrqPovLQjwYNhi",
		"16Uiu2HAmA4RpFMvce7JAMUgbfRGcLW4EqaYzHYefRNrSQauBwNUm",
	}, nil
}

func (cs *consensusServanMock) GetActiveProposerIDs() ([]string, error) {
	//TODO implement me
	panic("implement me")
}

func (cs *consensusServanMock) GetActiveValidatorIDs() ([]string, error) {
	//TODO implement me
	panic("implement me")
}

func (cs *consensusServanMock) GetTotalWeight() (uint64, error) {
	//TODO implement me
	panic("implement me")
}

func (cs *consensusServanMock) GetActiveExecutorsTotalWeight() (uint64, error) {
	return 30, nil
}

func (cs *consensusServanMock) GetActiveProposersTotalWeight() (uint64, error) {
	//TODO implement me
	panic("implement me")
}

func (cs *consensusServanMock) GetActiveValidatorsTotalWeight() (uint64, error) {
	//TODO implement me
	panic("implement me")
}

func (cs *consensusServanMock) GetNodeWeight(nodeID string) (uint64, error) {
	nwMap :=
		map[string]uint64{
			"16Uiu2HAmUnckEUPdYJ35h6DkUbNKDchbBDUd6heZi5DgBHkBJRCT": 10,
			"16Uiu2HAmKcwN8p9PZg4mnTSyruJpUNP4Crr9gfcrqPovLQjwYNhi": 20,
			"16Uiu2HAmA4RpFMvce7JAMUgbfRGcLW4EqaYzHYefRNrSQauBwNUm": 10,
		}

	return nwMap[nodeID], nil
}
