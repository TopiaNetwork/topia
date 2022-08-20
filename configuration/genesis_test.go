package configuration

import (
	"fmt"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"

	"github.com/TopiaNetwork/kyber/v3/pairing/bn256"
	"github.com/TopiaNetwork/kyber/v3/util/encoding"
	"github.com/TopiaNetwork/kyber/v3/util/key"

	tpchaintypes "github.com/TopiaNetwork/topia/chain/types"
	tpcmm "github.com/TopiaNetwork/topia/common"
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
		NetType: tpcmm.CurrentNetworkType,
		Epoch: &tpcmm.EpochInfo{
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

	suite := bn256.NewSuiteG2()

	exeDomainInfo1 := &tpcmm.NodeDomainInfo{
		ID:               tpcmm.CreateDomainID("exedomain1"),
		Type:             tpcmm.DomainType_Execute,
		ValidHeightStart: 1,
		ValidHeightEnd:   100000,
		ExeDomainData: &tpcmm.NodeExecuteDomain{
			Members: []string{
				"16Uiu2HAkvPb9xbeHsbDSS44xLY2ZWJvZtdUpcWBSAMuPBTx1eNnj",
				"16Uiu2HAm29UXXHcMbUXpeALDQuRppiRKxvCGnREN4XwKRTPgFV4Q",
				"16Uiu2HAm8r3jz6E9p1imUJh1wc4JprhHKDqBBZwu1gHsXSHJpM7d",
			},
		},
	}

	gData.GenesisExeDomain = append(gData.GenesisExeDomain, exeDomainInfo1)

	executorIDs := []string{
		"16Uiu2HAkvPb9xbeHsbDSS44xLY2ZWJvZtdUpcWBSAMuPBTx1eNnj",
		"16Uiu2HAm29UXXHcMbUXpeALDQuRppiRKxvCGnREN4XwKRTPgFV4Q",
		"16Uiu2HAm8r3jz6E9p1imUJh1wc4JprhHKDqBBZwu1gHsXSHJpM7d",
	}
	for i := 0; i < len(executorIDs); i++ {
		gData.GenesisNode = append(gData.GenesisNode, &tpcmm.NodeInfo{
			NodeID: executorIDs[i],
			Weight: 10,
			Role:   tpcmm.NodeRole_Executor,
			State:  tpcmm.NodeState_Active,
		})
	}

	proposerIDs := []string{
		"16Uiu2HAmRGnTWGLtCJaH7VZKrrAPuh4waQueTyUGNMJ1giktViaP",
		"16Uiu2HAmSxqKEEkjAHeUu2gpWUYdjEKbvz3t7S36haUtdkjPvgJu",
		"16Uiu2HAkzgCXoLsa6g1iSjBY3Lykeqg2EF44ZJLBGykRpAaA8uY7",
	}
	for i := 0; i < len(proposerIDs); i++ {
		keyPair := key.NewKeyPair(suite)
		dkgPriKey, _ := encoding.ScalarToStringHex(suite, keyPair.Private)
		fmt.Printf("keyPriKey %s: %s\n", proposerIDs[i], dkgPriKey)
		dkgPartPubKey, _ := encoding.PointToStringHex(suite, keyPair.Public)
		gData.GenesisNode = append(gData.GenesisNode, &tpcmm.NodeInfo{
			NodeID:        proposerIDs[i],
			Weight:        uint64(10 * (i + 1)),
			DKGPartPubKey: dkgPartPubKey,
			Role:          tpcmm.NodeRole_Proposer,
			State:         tpcmm.NodeState_Active,
		})
	}

	validatorIDs := []string{
		"16Uiu2HAmEjXsGN2yR6jXc1bPEiEs3o1aMu2apchx8taGArmKy7ja",
		"16Uiu2HAkxG2ZpVWub8xatogt7oP8cwu4pbbkxG1oQXrgNSembE5C",
		"16Uiu2HAmUpkpxGbEc4tUVPYmdaMeZS9VCbGiJV2mXFMiVCe71eDq",
		"16Uiu2HAmD57RgJxS7R4putawdjxz9KQqqtRnTDm4JoPeTjHTzN3R",
	}
	for i := 0; i < len(validatorIDs); i++ {
		keyPair := key.NewKeyPair(suite)
		dkgPriKey, _ := encoding.ScalarToStringHex(suite, keyPair.Private)
		fmt.Printf("keyPriKey %s: %s\n", validatorIDs[i], dkgPriKey)
		dkgPartPubKey, _ := encoding.PointToStringHex(suite, keyPair.Public)
		gData.GenesisNode = append(gData.GenesisNode, &tpcmm.NodeInfo{
			NodeID:        validatorIDs[i],
			Weight:        10,
			DKGPartPubKey: dkgPartPubKey,
			Role:          tpcmm.NodeRole_Validator,
			State:         tpcmm.NodeState_Active,
		})
	}

	err := gData.Save("./genesis.json")
	assert.Equal(t, nil, err)
	gData.Load()
	assert.Equal(t, nil, err)
}
