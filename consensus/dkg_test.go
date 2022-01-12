package consensus

import (
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/TopiaNetwork/kyber/v3/pairing/bn256"

	tplog "github.com/TopiaNetwork/topia/log"
	tplogcmm "github.com/TopiaNetwork/topia/log/common"
)

func TestCreateGenerator(t *testing.T) {
	log, _ := tplog.CreateMainLogger(tplogcmm.InfoLevel, tplog.DefaultLogFormat, tplog.DefaultLogOutput, "")
	suite := bn256.NewSuiteG2()

	creatKeyPairs(suite, 7)
	dkgCrypt := newDKGCrypt(log, 0, 10, suite, initPrivKeys[0], initPubKeys, 5, 7)

	err := dkgCrypt.createGenerator()
	require.Equal(t, nil, err)

	deals, _ := dkgCrypt.getDeals()
	require.Equal(t, 6, len(deals))
	require.Equal(t, 6, dkgCrypt.dkGenerator.ExpectedDeals())
	require.NotEqual(t, true, dkgCrypt.dkGenerator.Certified())
}
