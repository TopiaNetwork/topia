package consensus

import (
	"math/big"
	"sort"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestComputeVRF(t *testing.T) {
	crypt := &CryptServiceMock{}

	priKey, pubKey, _ := crypt.GeneratePriPubKey()

	addr, _ := crypt.CreateAddress(pubKey)

	selector := newProposerSelectorPoiss(crypt)

	vrfInputData := "TestSelectProposer1"

	vrfProof, err := selector.ComputeVRF(priKey, []byte(vrfInputData))
	require.Equal(t, nil, err)
	require.NotEqual(t, nil, vrfProof)

	isOk, err := crypt.Verify(addr, []byte(vrfInputData), vrfProof)
	require.Equal(t, nil, err)
	require.Equal(t, true, isOk)
}

func TestSelectProposer(t *testing.T) {
	crypt := &CryptServiceMock{}

	priKey, pubKey, _ := crypt.GeneratePriPubKey()

	addr, _ := crypt.CreateAddress(pubKey)

	selector := newProposerSelectorPoiss(crypt)

	vrfInputData := "TestSelectProposer1"

	vrfProof, err := selector.ComputeVRF(priKey, []byte(vrfInputData))
	require.Equal(t, nil, err)
	require.NotEqual(t, nil, vrfProof)

	isOk, err := crypt.Verify(addr, []byte(vrfInputData), vrfProof)
	require.Equal(t, nil, err)
	require.Equal(t, true, isOk)

	var maxPris []*big.Int

	pCount0 := selector.SelectProposer(vrfProof, big.NewInt(20), big.NewInt(30))
	if pCount0 > 0 {
		maxPris = append(maxPris, new(big.Int).SetBytes(selector.MaxPriority(vrfProof, pCount0)))
	}
	t.Logf("pCount0=%d, vrfProof=%v, weight proportion 20/30", pCount0, vrfProof)

	pCount1 := selector.SelectProposer(vrfProof, big.NewInt(30), big.NewInt(30))
	if pCount1 > 0 {
		maxPris = append(maxPris, new(big.Int).SetBytes(selector.MaxPriority(vrfProof, pCount1)))
	}
	t.Logf("pCount0=%d, vrfProof=%v, weight proportion 30/30", pCount1, vrfProof)

	pCount2 := selector.SelectProposer(vrfProof, big.NewInt(30), big.NewInt(30))
	if pCount2 > 0 {
		maxPris = append(maxPris, new(big.Int).SetBytes(selector.MaxPriority(vrfProof, pCount2)))
	}
	t.Logf("pCount2=%d, vrfProof=%v, weight proportion 30/30", pCount2, vrfProof)

	pCount3 := selector.SelectProposer(vrfProof, big.NewInt(10), big.NewInt(100))
	if pCount3 > 0 {
		maxPris = append(maxPris, new(big.Int).SetBytes(selector.MaxPriority(vrfProof, pCount3)))
	}
	t.Logf("pCount3=%d, vrfProof=%v, weight proportion 10/100", pCount3, vrfProof)

	pCount4 := selector.SelectProposer(vrfProof, big.NewInt(5), big.NewInt(100))
	if pCount4 > 0 {
		maxPris = append(maxPris, new(big.Int).SetBytes(selector.MaxPriority(vrfProof, pCount4)))
	}
	t.Logf("pCount4=%d, vrfProof=%v, weight proportion 5/100", pCount4, vrfProof)

	t.Logf("Before max pri sort: %v", maxPris)
	sort.Slice(maxPris, func(i, j int) bool {
		return maxPris[i].Cmp(maxPris[j]) < 0
	})
	t.Logf("After max pri sort: %v", maxPris)

}
