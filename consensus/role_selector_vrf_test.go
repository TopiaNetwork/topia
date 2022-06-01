package consensus

import (
	"testing"

	"github.com/stretchr/testify/assert"

	tplog "github.com/TopiaNetwork/topia/log"
	tplogcmm "github.com/TopiaNetwork/topia/log/common"
)

func TestSelect(t *testing.T) {
	log, _ := tplog.CreateMainLogger(tplogcmm.InfoLevel, tplog.DefaultLogFormat, tplog.DefaultLogOutput, "")
	cryptService := &CryptServiceMock{}
	servant := &consensusServanMock{}
	priKey1, _, _ := cryptService.GeneratePriPubKey()
	priKey2, _, _ := cryptService.GeneratePriPubKey()
	priKey3, _, _ := cryptService.GeneratePriPubKey()

	roleSel := newLeaderSelectorVRF(log, "", cryptService)

	t.Log("Node 1:")
	for i := 0; i < 5; i++ {
		canInfo1, _, err := roleSel.Select(RoleSelector_ExecutionLauncher, 1, priKey1, servant, 1)
		assert.Equal(t, nil, err)
		assert.Equal(t, 1, len(canInfo1))

		t.Logf("canInfo%d=%v", i+1, canInfo1[0])
	}

	t.Log("Node 2:")
	for i := 0; i < 5; i++ {
		canInfo1, _, err := roleSel.Select(RoleSelector_ExecutionLauncher, 2, priKey2, servant, 1)
		assert.Equal(t, nil, err)
		assert.Equal(t, 1, len(canInfo1))

		t.Logf("canInfo%d=%v", i+1, canInfo1[0])
	}

	t.Log("Node 3:")
	for i := 0; i < 5; i++ {
		canInfo1, _, err := roleSel.Select(RoleSelector_ExecutionLauncher, 3, priKey3, servant, 1)
		assert.Equal(t, nil, err)
		assert.Equal(t, 1, len(canInfo1))

		t.Logf("canInfo%d=%v", i+1, canInfo1[0])
	}
}
