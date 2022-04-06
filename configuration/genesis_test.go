package configuration

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"

	"github.com/TopiaNetwork/topia/chain"
	tpchaintypes "github.com/TopiaNetwork/topia/chain/types"
)

func TestGenerateGenesisData(t *testing.T) {
	timeStamp := uint64(time.Now().UnixNano())

	block := &tpchaintypes.Block{
		Head: &tpchaintypes.BlockHead{
			ChainID:   []byte("TestTopia"),
			Version:   1,
			Height:    1,
			Epoch:     0,
			Round:     1,
			TimeStamp: timeStamp,
		},
	}

	blockHashBytes, _ := block.HashBytes()

	gData := &GenesisData{
		Epon: &chain.EpochInfo{
			Epoch:          0,
			StartTimeStamp: timeStamp,
			StartHeight:    1,
		},
		Block: block,
		BlockResult: &tpchaintypes.BlockResult{
			Head: &tpchaintypes.BlockResultHead{
				BlockHash: blockHashBytes,
				Status:    tpchaintypes.BlockResultHead_OK,
			},
		},
	}

	err := gData.Save("./genesis.json")
	assert.Equal(t, nil, err)
}
