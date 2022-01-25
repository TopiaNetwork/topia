package consensus

import (
	"math/big"
	"testing"

	"github.com/stretchr/testify/require"

	tpcmm "github.com/TopiaNetwork/topia/common"
)

func TestComputeVRF(t *testing.T) {
	crypt := &CryptServiceMock{}

	priKey, pubKey, _ := crypt.GeneratePriPubKey()

	selector := newProposerSelectorPoiss(crypt, tpcmm.NewBlake2bHasher(0))

	vrfInputData := "TestSelectProposer1"

	vrfProof, err := selector.ComputeVRF(priKey, []byte(vrfInputData))
	require.Equal(t, nil, err)
	require.NotEqual(t, nil, vrfProof)

	isOk, err := crypt.Verify(pubKey, []byte(vrfInputData), vrfProof)
	require.Equal(t, nil, err)
	require.Equal(t, true, isOk)
}

func TestSelectProposer(t *testing.T) {
	crypt := &CryptServiceMock{}

	priKey, pubKey, _ := crypt.GeneratePriPubKey()

	selector := newProposerSelectorPoiss(crypt, tpcmm.NewBlake2bHasher(0))

	vrfInputData := "TestSelectProposer1"

	vrfProof, err := selector.ComputeVRF(priKey, []byte(vrfInputData))
	require.Equal(t, nil, err)
	require.NotEqual(t, nil, vrfProof)

	isOk, err := crypt.Verify(pubKey, []byte(vrfInputData), vrfProof)
	require.Equal(t, nil, err)
	require.Equal(t, true, isOk)

	pCount0 := selector.SelectProposer(vrfProof, big.NewInt(20), big.NewInt(30))
	t.Logf("pCount0=%d, weight proportion 20/30", pCount0)

	pCount1 := selector.SelectProposer(vrfProof, big.NewInt(30), big.NewInt(30))
	t.Logf("pCount1=%d, weight proportion 30/30", pCount1)

	pCount2 := selector.SelectProposer(vrfProof, big.NewInt(30), big.NewInt(30))
	t.Logf("pCount2=%d, weight proportion 30/30", pCount2)

	pCount3 := selector.SelectProposer(vrfProof, big.NewInt(1), big.NewInt(100))
	t.Logf("pCount3=%d, weight proportion 1/100", pCount3)

	pCount4 := selector.SelectProposer(vrfProof, big.NewInt(1), big.NewInt(100))
	t.Logf("pCount4=%d, weight proportion 1/100", pCount4)
}
