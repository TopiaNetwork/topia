package consensus

import (
	"encoding/binary"
	"errors"
	"math/big"
	"sort"
	"strings"

	tpcmm "github.com/TopiaNetwork/topia/common"
	tpcrt "github.com/TopiaNetwork/topia/crypt"
	tpcrtypes "github.com/TopiaNetwork/topia/crypt/types"
	tplog "github.com/TopiaNetwork/topia/log"
)

const uint64Mask = uint64(0x7FFFFFFFFFFFFFFF)

var divider *big.Int

type RoleSelector byte

const (
	RoleSelector_Unknown = iota
	RoleSelector_Proposer
	RoleSelector_VoteCollector
)

func init() {
	divider = big.NewInt(int64(uint64Mask))
	divider.Add(divider, big.NewInt(1))
}

type candidateInfo struct {
	nodeID string
	weight uint64
}

type roleSelectorVRF struct {
	log     tplog.Logger
	crypt   tpcrt.CryptService
	csState consensusServant
}

func newLeaderSelectorVRF(log tplog.Logger, crypt tpcrt.CryptService, csState consensusServant) *roleSelectorVRF {
	return &roleSelectorVRF{
		log:     log,
		crypt:   crypt,
		csState: csState,
	}
}

func (selector *roleSelectorVRF) ComputeVRF(priKey tpcrtypes.PrivateKey, data []byte) ([]byte, error) {
	return selector.crypt.Sign(priKey, data)
}

// SplitMix64
// http://xoshiro.di.unimi.it/splitmix64.c
//
// The PRNG used for this random selection:
//   1. must be deterministic.
//   2. should easily portable, independent of language or library
//   3. is not necessary to keep a long period like MT, since there aren't many random numbers to generate and
//      we expect a certain amount of randomness in the seed.
func (selector *roleSelectorVRF) nextRandom(rand *uint64) uint64 {
	*rand += uint64(0x9e3779b97f4a7c15)
	var z = *rand
	z = (z ^ (z >> 30)) * 0xbf58476d1ce4e5b9
	z = (z ^ (z >> 27)) * 0x94d049bb133111eb
	return z ^ (z >> 31)
}

func (selector *roleSelectorVRF) thresholdValue(seed *uint64, totalWeight uint64) uint64 {
	a := new(big.Int).SetUint64(selector.nextRandom(seed) & uint64Mask)
	tWeight := new(big.Int).SetUint64(totalWeight)
	a.Mul(a, tWeight)
	a.Div(a, divider)
	return a.Uint64()
}

func (selector *roleSelectorVRF) sort(canInfos []*candidateInfo) []*candidateInfo {
	cans := make([]*candidateInfo, len(canInfos))
	copy(cans, canInfos)
	sort.Slice(cans, func(i, j int) bool {
		if cans[i].weight != cans[j].weight {
			return cans[i].weight > cans[j].weight
		}
		return strings.Compare(cans[i].nodeID, cans[j].nodeID) < 0
	})
	return cans
}

func (selector *roleSelectorVRF) hashToSeed(hash []byte) uint64 {
	for len(hash) < 8 {
		hash = append(hash, byte(0))
	}
	return binary.LittleEndian.Uint64(hash[:8])
}

func (selector *roleSelectorVRF) makeVRFHash(role RoleSelector, epoch uint64, round uint64, vrfProof []byte) []byte {
	b := make([]byte, 24)
	binary.LittleEndian.PutUint64(b, uint64(role))
	binary.LittleEndian.PutUint64(b[8:16], epoch)
	binary.LittleEndian.PutUint64(b[16:24], round)

	hasher := tpcmm.NewBlake2bHasher(0)

	if _, err := hasher.Writer().Write(vrfProof); err != nil {
		panic(err)
	}
	if _, err := hasher.Writer().Write(b[:8]); err != nil {
		panic(err)
	}
	if _, err := hasher.Writer().Write(b[8:16]); err != nil {
		panic(err)
	}
	if _, err := hasher.Writer().Write(b[8:24]); err != nil {
		panic(err)
	}

	return hasher.Bytes()
}

func (selector *roleSelectorVRF) getVrfInputData(role RoleSelector, roundInfo *RoundInfo) ([]byte, error) {
	hasher := tpcmm.NewBlake2bHasher(0)

	if err := binary.Write(hasher.Writer(), binary.BigEndian, role); err != nil {
		return nil, err
	}
	if err := binary.Write(hasher.Writer(), binary.BigEndian, roundInfo.Epoch); err != nil {
		return nil, err
	}
	if err := binary.Write(hasher.Writer(), binary.BigEndian, roundInfo.CurRoundNum); err != nil {
		return nil, err
	}

	csProofBytes, err := roundInfo.Proof.Marshal()
	if err != nil {
		return nil, err
	}
	if _, err = hasher.Writer().Write(csProofBytes); err != nil {
		return nil, err
	}

	return hasher.Bytes(), nil
}

func (selector *roleSelectorVRF) getCandidateInfos() ([]*candidateInfo, error) {
	csNodes, err := selector.csState.GetAllConsensusNodes()
	if err != nil {
		selector.log.Errorf("Can't get all consensus nodes: %v", err)
		return nil, err
	}

	var canInfos []*candidateInfo
	for _, nodeId := range csNodes {
		nodeWeight, err := selector.csState.GetNodeWeight(nodeId)
		if err != nil {
			selector.log.Errorf("Can't get node weight: %v", err)
			return nil, err
		}
		canInfo := &candidateInfo{
			nodeID: nodeId,
			weight: nodeWeight,
		}
		canInfos = append(canInfos, canInfo)
	}

	return canInfos, nil
}

func (selector *roleSelectorVRF) Select(role RoleSelector,
	roundInfo *RoundInfo,
	priKey tpcrtypes.PrivateKey,
	count int) ([]*candidateInfo, []byte, error) {
	thresholdVals := make([]uint64, count)
	totalWeight, err := selector.csState.GetChainTotalWeight()
	if err != nil {
		return nil, nil, err
	}

	canInfos, err := selector.getCandidateInfos()
	if err != nil {
		return nil, nil, err
	}

	vrfInputData, err := selector.getVrfInputData(role, roundInfo)
	if err != nil {
		selector.log.Errorf("Can't get vrf inputing data: epoch =%d, new round=%d, err=%v", roundInfo.Epoch, roundInfo.CurRoundNum, err)
		return nil, nil, err
	}

	vrfProof, err := selector.ComputeVRF(priKey, vrfInputData)
	if err != nil {
		return nil, nil, err
	}

	vrfHash := selector.makeVRFHash(role, roundInfo.Epoch, roundInfo.CurRoundNum, vrfProof)
	seed := selector.hashToSeed(vrfHash)

	for i := 0; i < count; i++ {
		thresholdVals[i] = selector.thresholdValue(&seed, totalWeight)
	}
	sort.Slice(thresholdVals, func(i, j int) bool { return thresholdVals[i] < thresholdVals[j] })

	cans := selector.sort(canInfos)

	cansResult := make([]*candidateInfo, count)
	cumulativeWeight := uint64(0)
	undrawn := 0
	for _, can := range cans {
		if thresholdVals[undrawn] < (cumulativeWeight + can.weight) {
			cansResult[undrawn] = can
			undrawn++
			if undrawn == len(cansResult) {
				return cansResult, vrfProof, nil
			}
		}
		cumulativeWeight = cumulativeWeight + can.weight
	}

	return cansResult, vrfProof, errors.New("Invalid parameters")
}
