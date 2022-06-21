package vrf

import (
	tpchaintypes "github.com/TopiaNetwork/topia/chain/types"
	"github.com/TopiaNetwork/topia/consensus"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"

	tplog "github.com/TopiaNetwork/topia/log"
	tplogcmm "github.com/TopiaNetwork/topia/log/common"
)

func TestSelect(t *testing.T) {
	log, _ := tplog.CreateMainLogger(tplogcmm.InfoLevel, tplog.DefaultLogFormat, tplog.DefaultLogOutput, "")
	latestBlock := &tpchaintypes.Block{
		Head: &tpchaintypes.BlockHead{
			ChainID:   []byte("TestTopia"),
			Version:   1,
			Height:    1,
			Epoch:     0,
			Round:     1,
			TimeStamp: uint64(time.Now().UnixNano()),
		},
	}

	cryptService := &consensus.CryptServiceMock{}
	epochService := &consensus.EpochServiceMock{}
	priKey1, _, _ := cryptService.GeneratePriPubKey()
	priKey2, _, _ := cryptService.GeneratePriPubKey()
	priKey3, _, _ := cryptService.GeneratePriPubKey()

	roleSel := NewLeaderSelectorVRF(log, "", cryptService)

	t.Log("Node 1:")
	for i := 0; i < 5; i++ {
		canInfo1, _, err := roleSel.Select(RoleSelector_ExecutionLauncher, 1, priKey1, latestBlock, epochService, 1)
		assert.Equal(t, nil, err)
		assert.Equal(t, 1, len(canInfo1))

		t.Logf("canInfo%d=%v", i+1, canInfo1[0])
	}

	t.Log("Node 2:")
	for i := 0; i < 5; i++ {
		canInfo1, _, err := roleSel.Select(RoleSelector_ExecutionLauncher, 2, priKey2, latestBlock, epochService, 1)
		assert.Equal(t, nil, err)
		assert.Equal(t, 1, len(canInfo1))

		t.Logf("canInfo%d=%v", i+1, canInfo1[0])
	}

	t.Log("Node 3:")
	for i := 0; i < 5; i++ {
		canInfo1, _, err := roleSel.Select(RoleSelector_ExecutionLauncher, 3, priKey3, latestBlock, epochService, 1)
		assert.Equal(t, nil, err)
		assert.Equal(t, 1, len(canInfo1))

		t.Logf("canInfo%d=%v", i+1, canInfo1[0])
	}
}
