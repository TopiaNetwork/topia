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
	priKey, _, _ := cryptService.GeneratePriPubKey()

	roleSel := newLeaderSelectorVRF(log, cryptService)

	for i := 0; i < 5; i++ {
		canInfo1, _, err := roleSel.Select(RoleSelector_ExecutionLauncher, 1, priKey, servant, 1)
		assert.Equal(t, nil, err)
		assert.Equal(t, 1, len(canInfo1))

		t.Logf("canInfo%d=%v", i+1, canInfo1[0])
	}
}
