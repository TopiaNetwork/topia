package consensus

import (
	"encoding/binary"
	"errors"
	"fmt"
	tpchaintypes "github.com/TopiaNetwork/topia/chain/types"
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
	RoleSelector_Unknown RoleSelector = iota
	RoleSelector_ExecutionLauncher
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
	log    tplog.Logger
	nodeID string
	crypt  tpcrt.CryptService
}

func newLeaderSelectorVRF(log tplog.Logger, nodeID string, crypt tpcrt.CryptService) *roleSelectorVRF {
	return &roleSelectorVRF{
		log:    log,
		nodeID: nodeID,
		crypt:  crypt,
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
	return binary.BigEndian.Uint64(hash[:8])
}

func (selector *roleSelectorVRF) makeVRFHash(role RoleSelector, epoch uint64, round uint64, vrfProof []byte) []byte {
	b := make([]byte, 24)
	binary.BigEndian.PutUint64(b, uint64(role))
	binary.BigEndian.PutUint64(b[8:16], epoch)
	binary.BigEndian.PutUint64(b[16:24], round)

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

func (selector *roleSelectorVRF) getVrfInputData(role RoleSelector, epoch uint64, height uint64, csProofBytes []byte, stateVersion uint64) ([]byte, error) {
	hasher := tpcmm.NewBlake2bHasher(0)

	if err := binary.Write(hasher.Writer(), binary.BigEndian, role); err != nil {
		return nil, err
	}
	if err := binary.Write(hasher.Writer(), binary.BigEndian, epoch); err != nil {
		return nil, err
	}
	/*
		if err := binary.Write(hasher.Writer(), binary.BigEndian, height); err != nil {
			return nil, err
		}
	*/
	if err := binary.Write(hasher.Writer(), binary.BigEndian, stateVersion); err != nil {
		return nil, err
	}
	/*
		if _, err := hasher.Writer().Write(csProofBytes); err != nil {
			return nil, err
		}
	*/

	return hasher.Bytes(), nil
}

func (selector *roleSelectorVRF) getCandidateInfos(avtiveNodeID []string, epochService EpochService) ([]*candidateInfo, error) {
	var canInfos []*candidateInfo
	for _, nodeId := range avtiveNodeID {
		nodeWeight, err := epochService.GetNodeWeight(nodeId)
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
	stateVersion uint64,
	priKey tpcrtypes.PrivateKey,
	latestBlock *tpchaintypes.Block,
	epochService EpochService,
	count int) ([]*candidateInfo, []byte, error) {
	thresholdVals := make([]uint64, count)

	err := error(nil)
	var totalActiveWeight uint64
	var avtiveNodeID []string

	selector.log.Infof("Enter Select: state version %d, self node %s", stateVersion, selector.nodeID)

	switch role {
	case RoleSelector_ExecutionLauncher:
		{
			totalActiveWeight = epochService.GetActiveExecutorsTotalWeight()
			avtiveNodeID = epochService.GetActiveExecutorIDs()
		}
	case RoleSelector_VoteCollector:
		{
			totalActiveWeight = epochService.GetActiveValidatorsTotalWeight()
			avtiveNodeID = epochService.GetActiveValidatorIDs()
		}
	default:
		return nil, nil, fmt.Errorf("Invalid role %s", role.String())
	}

	sort.Strings(avtiveNodeID)

	canInfos, err := selector.getCandidateInfos(avtiveNodeID, epochService)
	if err != nil {
		return nil, nil, err
	}

	eponInfo := epochService.GetLatestEpoch()

	csProof := &ConsensusProof{
		ParentBlockHash: latestBlock.Head.ParentBlockHash,
		Height:          latestBlock.Head.Height,
		AggSign:         latestBlock.Head.VoteAggSignature,
	}
	csProofBytes, err := csProof.Marshal()
	if err != nil {
		return nil, nil, err
	}

	selector.log.Infof("Vrf input data: role %s, epoch %d, state version %d, self node %s", role.String(), eponInfo.Epoch, stateVersion, selector.nodeID)

	vrfInputData, err := selector.getVrfInputData(role, eponInfo.Epoch, latestBlock.Head.Height, csProofBytes, stateVersion)
	if err != nil {
		selector.log.Errorf("Can't get vrf inputting data: epoch=%d, height=%d, err=%v", eponInfo.Epoch, latestBlock.Head.Height, err)
		return nil, nil, err
	}

	vrfProof, err := selector.ComputeVRF(priKey, vrfInputData)
	if err != nil {
		return nil, nil, err
	}

	vrfHash := selector.makeVRFHash(role, eponInfo.Epoch, stateVersion, vrfInputData)
	seed := selector.hashToSeed(vrfHash)

	for i := 0; i < count; i++ {
		thresholdVals[i] = selector.thresholdValue(&seed, totalActiveWeight)
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

func (r RoleSelector) String() string {
	switch r {
	case RoleSelector_ExecutionLauncher:
		return "ExecutionLauncher"
	case RoleSelector_VoteCollector:
		return "VoteCollector"
	default:
		return "Unknown"
	}
}

func (r RoleSelector) Value(roleSelector string) RoleSelector {
	switch roleSelector {
	case "ExecutionLauncher":
		return RoleSelector_ExecutionLauncher
	case "VoteCollector":
		return RoleSelector_VoteCollector
	default:
		return RoleSelector_Unknown
	}
}
