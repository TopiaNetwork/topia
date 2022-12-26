package configuration

import (
	"encoding/hex"
	"fmt"
	"math/big"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"

	"github.com/TopiaNetwork/kyber/v3/pairing/bn256"
	"github.com/TopiaNetwork/kyber/v3/util/encoding"
	"github.com/TopiaNetwork/kyber/v3/util/key"

	tpacc "github.com/TopiaNetwork/topia/account"
	tpchaintypes "github.com/TopiaNetwork/topia/chain/types"
	tpcmm "github.com/TopiaNetwork/topia/common"
	tpcrtypes "github.com/TopiaNetwork/topia/crypt/types"
	"github.com/TopiaNetwork/topia/currency"
)

/*
Account priKey 35a64a1b110f499e0847b2ce87480063f81ead7bc10c65202d7cadf87501c11ce91b83bc5b42918daa92e38691f4f976a30401f0f7d41a6d275d4370c80180ed
pubKey e91b83bc5b42918daa92e38691f4f976a30401f0f7d41a6d275d4370c80180ed
addr t34afywhp2rbry2bsm3giki4wzg1ljrthj56bos2kwcuoxomtojivxnuxwy3
Account priKey a9f44ca2064c9d027db3e28459e7128431e074adeb8a190601dcd3ab3606184910c443c7853f72336e1d66feb10938dc66649628c167f323ebde75b9707cf005
pubKey 10c443c7853f72336e1d66feb10938dc66649628c167f323ebde75b9707cf005
addr t3pipawl3nw4zie2j4d26cppky2lqeknlrynq6er6c2z12m3i35tp4tztrca
*/

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
		InitAccounts: make(map[tpcrtypes.Address]*tpacc.Account),
	}

	cryptService := &CryptServiceMock{}

	i := 0
	for i < 2 {
		priKey, pubKey, _ := cryptService.GeneratePriPubKey()
		addr, _ := cryptService.CreateAddress(pubKey)
		accNew := tpacc.NewDefaultAccount(addr)
		accNew.Balances[currency.TokenSymbol_Native] = big.NewInt(10000000000)

		fmt.Printf("Account priKey %s\npubKey %s\naddr %s\n", hex.EncodeToString(priKey), hex.EncodeToString(pubKey), addr)
		gData.InitAccounts[addr] = accNew
		i++
	}

	suite := bn256.NewSuiteG2()

	gData.GenesisNode = make(map[string]*tpcmm.NodeInfo)
	gData.SeedPeersMap = make(map[string][]*SeedPeer)

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

	gData.SeedPeersMap[exeDomainInfo1.ID] = []*SeedPeer{
		{tpcmm.NodeRole_Executor, "/ip4/192.168.3.35/tcp/41000/p2p/16Uiu2HAkvPb9xbeHsbDSS44xLY2ZWJvZtdUpcWBSAMuPBTx1eNnj"},
		{tpcmm.NodeRole_Executor, "/ip4/192.168.3.36/tcp/41001/p2p/16Uiu2HAm29UXXHcMbUXpeALDQuRppiRKxvCGnREN4XwKRTPgFV4Q"},
		{tpcmm.NodeRole_Executor, "/ip4/192.168.3.37/tcp/41002/p2p/16Uiu2HAm8r3jz6E9p1imUJh1wc4JprhHKDqBBZwu1gHsXSHJpM7d"},
	}

	gData.SeedPeersMap["consensus"] = append(gData.SeedPeersMap["consensus"],
		&SeedPeer{tpcmm.NodeRole_Executor, "/ip4/192.168.3.35/tcp/41000/p2p/16Uiu2HAkvPb9xbeHsbDSS44xLY2ZWJvZtdUpcWBSAMuPBTx1eNnj"})
	gData.SeedPeersMap["consensus"] = append(gData.SeedPeersMap["consensus"],
		&SeedPeer{tpcmm.NodeRole_Executor, "/ip4/192.168.3.36/tcp/41001/p2p/16Uiu2HAm29UXXHcMbUXpeALDQuRppiRKxvCGnREN4XwKRTPgFV4Q"})
	gData.SeedPeersMap["consensus"] = append(gData.SeedPeersMap["consensus"],
		&SeedPeer{tpcmm.NodeRole_Executor, "/ip4/192.168.3.37/tcp/41002/p2p/16Uiu2HAm8r3jz6E9p1imUJh1wc4JprhHKDqBBZwu1gHsXSHJpM7d"})

	exeDomainInfo2 := &tpcmm.NodeDomainInfo{
		ID:               tpcmm.CreateDomainID("exedomain2"),
		Type:             tpcmm.DomainType_Execute,
		ValidHeightStart: 1,
		ValidHeightEnd:   100000,
		ExeDomainData: &tpcmm.NodeExecuteDomain{
			Members: []string{
				"16Uiu2HAm63TystBV4hVt2s8ibvbLamcCtLjSSZBkpcET9df91dRy",
				"16Uiu2HAmL35p4tw2kHMbtKkrfKNc2JoWeWovfWXPFzDPr5E2qHqE",
				"16Uiu2HAmFt98Z258smgyrmzmfyyfwSqZxJZTZ2N5wvqABnDELrca",
			},
		},
	}

	gData.SeedPeersMap[exeDomainInfo2.ID] = []*SeedPeer{
		{tpcmm.NodeRole_Executor, "/ip4/192.168.3.38/tcp/41010/p2p/16Uiu2HAm63TystBV4hVt2s8ibvbLamcCtLjSSZBkpcET9df91dRy"},
		{tpcmm.NodeRole_Executor, "/ip4/192.168.3.39/tcp/41011/p2p/16Uiu2HAmL35p4tw2kHMbtKkrfKNc2JoWeWovfWXPFzDPr5E2qHqE"},
		{tpcmm.NodeRole_Executor, "/ip4/192.168.3.40/tcp/41022/p2p/16Uiu2HAmFt98Z258smgyrmzmfyyfwSqZxJZTZ2N5wvqABnDELrca"},
	}

	gData.SeedPeersMap["consensus"] = append(gData.SeedPeersMap["consensus"],
		&SeedPeer{tpcmm.NodeRole_Executor, "/ip4/192.168.3.38/tcp/41010/p2p/16Uiu2HAm63TystBV4hVt2s8ibvbLamcCtLjSSZBkpcET9df91dRy"})
	gData.SeedPeersMap["consensus"] = append(gData.SeedPeersMap["consensus"],
		&SeedPeer{tpcmm.NodeRole_Executor, "/ip4/192.168.3.39/tcp/41011/p2p/16Uiu2HAmL35p4tw2kHMbtKkrfKNc2JoWeWovfWXPFzDPr5E2qHqE"})
	gData.SeedPeersMap["consensus"] = append(gData.SeedPeersMap["consensus"],
		&SeedPeer{tpcmm.NodeRole_Executor, "/ip4/192.168.3.40/tcp/41022/p2p/16Uiu2HAmFt98Z258smgyrmzmfyyfwSqZxJZTZ2N5wvqABnDELrca"})

	gData.GenesisExeDomain = append(gData.GenesisExeDomain, exeDomainInfo1)
	gData.GenesisExeDomain = append(gData.GenesisExeDomain, exeDomainInfo2)

	executorSeeds := []string{
		"topia0",
		"topia1",
		"topia2",
		"topia00",
		"topia11",
		"topia22",
	}
	executorIDs := []string{
		"16Uiu2HAkvPb9xbeHsbDSS44xLY2ZWJvZtdUpcWBSAMuPBTx1eNnj",
		"16Uiu2HAm29UXXHcMbUXpeALDQuRppiRKxvCGnREN4XwKRTPgFV4Q",
		"16Uiu2HAm8r3jz6E9p1imUJh1wc4JprhHKDqBBZwu1gHsXSHJpM7d",
		"16Uiu2HAm63TystBV4hVt2s8ibvbLamcCtLjSSZBkpcET9df91dRy",
		"16Uiu2HAmL35p4tw2kHMbtKkrfKNc2JoWeWovfWXPFzDPr5E2qHqE",
		"16Uiu2HAmFt98Z258smgyrmzmfyyfwSqZxJZTZ2N5wvqABnDELrca",
	}
	for i := 0; i < len(executorIDs); i++ {
		gData.GenesisNode[executorSeeds[i]] = &tpcmm.NodeInfo{
			NodeID: executorIDs[i],
			Weight: 10,
			Role:   tpcmm.NodeRole_Executor,
			State:  tpcmm.NodeState_Active,
		}
	}

	proposerSeeds := []string{
		"topia3",
		"topia4",
		"topia5",
	}
	proposerIDs := []string{
		"16Uiu2HAmRGnTWGLtCJaH7VZKrrAPuh4waQueTyUGNMJ1giktViaP",
		"16Uiu2HAmSxqKEEkjAHeUu2gpWUYdjEKbvz3t7S36haUtdkjPvgJu",
		"16Uiu2HAkzgCXoLsa6g1iSjBY3Lykeqg2EF44ZJLBGykRpAaA8uY7",
	}

	gData.SeedPeersMap[exeDomainInfo1.ID] = append(gData.SeedPeersMap[exeDomainInfo1.ID],
		&SeedPeer{tpcmm.NodeRole_Proposer, "/ip4/192.168.3.41/tcp/41003/p2p/16Uiu2HAmRGnTWGLtCJaH7VZKrrAPuh4waQueTyUGNMJ1giktViaP"})
	gData.SeedPeersMap[exeDomainInfo1.ID] = append(gData.SeedPeersMap[exeDomainInfo1.ID],
		&SeedPeer{tpcmm.NodeRole_Proposer, "/ip4/192.168.3.42/tcp/41004/p2p/16Uiu2HAmSxqKEEkjAHeUu2gpWUYdjEKbvz3t7S36haUtdkjPvgJu"})
	gData.SeedPeersMap[exeDomainInfo1.ID] = append(gData.SeedPeersMap[exeDomainInfo1.ID],
		&SeedPeer{tpcmm.NodeRole_Proposer, "/ip4/192.168.3.43/tcp/41005/p2p/16Uiu2HAkzgCXoLsa6g1iSjBY3Lykeqg2EF44ZJLBGykRpAaA8uY7"})

	gData.SeedPeersMap[exeDomainInfo2.ID] = append(gData.SeedPeersMap[exeDomainInfo2.ID],
		&SeedPeer{tpcmm.NodeRole_Proposer, "/ip4/192.168.3.41/tcp/41003/p2p/16Uiu2HAmRGnTWGLtCJaH7VZKrrAPuh4waQueTyUGNMJ1giktViaP"})
	gData.SeedPeersMap[exeDomainInfo2.ID] = append(gData.SeedPeersMap[exeDomainInfo2.ID],
		&SeedPeer{tpcmm.NodeRole_Proposer, "/ip4/192.168.3.42/tcp/41004/p2p/16Uiu2HAmSxqKEEkjAHeUu2gpWUYdjEKbvz3t7S36haUtdkjPvgJu"})
	gData.SeedPeersMap[exeDomainInfo2.ID] = append(gData.SeedPeersMap[exeDomainInfo2.ID],
		&SeedPeer{tpcmm.NodeRole_Proposer, "/ip4/192.168.3.43/tcp/41005/p2p/16Uiu2HAkzgCXoLsa6g1iSjBY3Lykeqg2EF44ZJLBGykRpAaA8uY7"})

	gData.SeedPeersMap["consensus"] = append(gData.SeedPeersMap["consensus"],
		&SeedPeer{tpcmm.NodeRole_Proposer, "/ip4/192.168.3.41/tcp/41003/p2p/16Uiu2HAmRGnTWGLtCJaH7VZKrrAPuh4waQueTyUGNMJ1giktViaP"})
	gData.SeedPeersMap["consensus"] = append(gData.SeedPeersMap["consensus"],
		&SeedPeer{tpcmm.NodeRole_Proposer, "/ip4/192.168.3.42/tcp/41004/p2p/16Uiu2HAmSxqKEEkjAHeUu2gpWUYdjEKbvz3t7S36haUtdkjPvgJu"})
	gData.SeedPeersMap["consensus"] = append(gData.SeedPeersMap["consensus"],
		&SeedPeer{tpcmm.NodeRole_Proposer, "/ip4/192.168.3.43/tcp/41005/p2p/16Uiu2HAkzgCXoLsa6g1iSjBY3Lykeqg2EF44ZJLBGykRpAaA8uY7"})

	for i := 0; i < len(proposerIDs); i++ {
		keyPair := key.NewKeyPair(suite)
		dkgPriKey, _ := encoding.ScalarToStringHex(suite, keyPair.Private)
		fmt.Printf("keyPriKey %s: %s\n", proposerIDs[i], dkgPriKey)
		dkgPartPubKey, _ := encoding.PointToStringHex(suite, keyPair.Public)
		gData.GenesisNode[proposerSeeds[i]] = &tpcmm.NodeInfo{
			NodeID:        proposerIDs[i],
			Weight:        uint64(10 * (i + 1)),
			DKGPartPubKey: dkgPartPubKey,
			Role:          tpcmm.NodeRole_Proposer,
			State:         tpcmm.NodeState_Active,
		}
	}

	validatorSeeds := []string{
		"topia6",
		"topia7",
		"topia8",
		"topia9",
	}
	validatorIDs := []string{
		"16Uiu2HAmEjXsGN2yR6jXc1bPEiEs3o1aMu2apchx8taGArmKy7ja",
		"16Uiu2HAkxG2ZpVWub8xatogt7oP8cwu4pbbkxG1oQXrgNSembE5C",
		"16Uiu2HAmUpkpxGbEc4tUVPYmdaMeZS9VCbGiJV2mXFMiVCe71eDq",
		"16Uiu2HAmD57RgJxS7R4putawdjxz9KQqqtRnTDm4JoPeTjHTzN3R",
	}

	gData.SeedPeersMap[exeDomainInfo1.ID] = append(gData.SeedPeersMap[exeDomainInfo1.ID],
		&SeedPeer{tpcmm.NodeRole_Validator, "/ip4/192.168.3.44/tcp/41006/p2p/16Uiu2HAmEjXsGN2yR6jXc1bPEiEs3o1aMu2apchx8taGArmKy7ja"})
	gData.SeedPeersMap[exeDomainInfo1.ID] = append(gData.SeedPeersMap[exeDomainInfo1.ID],
		&SeedPeer{tpcmm.NodeRole_Validator, "/ip4/192.168.3.45/tcp/41007/p2p/16Uiu2HAkxG2ZpVWub8xatogt7oP8cwu4pbbkxG1oQXrgNSembE5C"})
	gData.SeedPeersMap[exeDomainInfo1.ID] = append(gData.SeedPeersMap[exeDomainInfo1.ID],
		&SeedPeer{tpcmm.NodeRole_Validator, "/ip4/192.168.3.46/tcp/41008/p2p/16Uiu2HAmUpkpxGbEc4tUVPYmdaMeZS9VCbGiJV2mXFMiVCe71eDq"})
	gData.SeedPeersMap[exeDomainInfo1.ID] = append(gData.SeedPeersMap[exeDomainInfo1.ID],
		&SeedPeer{tpcmm.NodeRole_Validator, "/ip4/192.168.3.47/tcp/41009/p2p/16Uiu2HAmD57RgJxS7R4putawdjxz9KQqqtRnTDm4JoPeTjHTzN3R"})

	gData.SeedPeersMap[exeDomainInfo2.ID] = append(gData.SeedPeersMap[exeDomainInfo2.ID],
		&SeedPeer{tpcmm.NodeRole_Validator, "/ip4/192.168.3.44/tcp/41006/p2p/16Uiu2HAmEjXsGN2yR6jXc1bPEiEs3o1aMu2apchx8taGArmKy7ja"})
	gData.SeedPeersMap[exeDomainInfo2.ID] = append(gData.SeedPeersMap[exeDomainInfo2.ID],
		&SeedPeer{tpcmm.NodeRole_Validator, "/ip4/192.168.3.45/tcp/41007/p2p/16Uiu2HAkxG2ZpVWub8xatogt7oP8cwu4pbbkxG1oQXrgNSembE5C"})
	gData.SeedPeersMap[exeDomainInfo2.ID] = append(gData.SeedPeersMap[exeDomainInfo2.ID],
		&SeedPeer{tpcmm.NodeRole_Validator, "/ip4/192.168.3.46/tcp/41008/p2p/16Uiu2HAmUpkpxGbEc4tUVPYmdaMeZS9VCbGiJV2mXFMiVCe71eDq"})
	gData.SeedPeersMap[exeDomainInfo2.ID] = append(gData.SeedPeersMap[exeDomainInfo2.ID],
		&SeedPeer{tpcmm.NodeRole_Validator, "/ip4/192.168.3.47/tcp/41009/p2p/16Uiu2HAmD57RgJxS7R4putawdjxz9KQqqtRnTDm4JoPeTjHTzN3R"})

	gData.SeedPeersMap["consensus"] = append(gData.SeedPeersMap["consensus"],
		&SeedPeer{tpcmm.NodeRole_Validator, "/ip4/192.168.3.44/tcp/41006/p2p/16Uiu2HAmEjXsGN2yR6jXc1bPEiEs3o1aMu2apchx8taGArmKy7ja"})
	gData.SeedPeersMap["consensus"] = append(gData.SeedPeersMap["consensus"],
		&SeedPeer{tpcmm.NodeRole_Validator, "/ip4/192.168.3.45/tcp/41007/p2p/16Uiu2HAkxG2ZpVWub8xatogt7oP8cwu4pbbkxG1oQXrgNSembE5C"})
	gData.SeedPeersMap["consensus"] = append(gData.SeedPeersMap["consensus"],
		&SeedPeer{tpcmm.NodeRole_Validator, "/ip4/192.168.3.46/tcp/41008/p2p/16Uiu2HAmUpkpxGbEc4tUVPYmdaMeZS9VCbGiJV2mXFMiVCe71eDq"})
	gData.SeedPeersMap["consensus"] = append(gData.SeedPeersMap["consensus"],
		&SeedPeer{tpcmm.NodeRole_Validator, "/ip4/192.168.3.47/tcp/41009/p2p/16Uiu2HAmD57RgJxS7R4putawdjxz9KQqqtRnTDm4JoPeTjHTzN3R"})

	for i := 0; i < len(validatorIDs); i++ {
		keyPair := key.NewKeyPair(suite)
		dkgPriKey, _ := encoding.ScalarToStringHex(suite, keyPair.Private)
		fmt.Printf("keyPriKey %s: %s\n", validatorIDs[i], dkgPriKey)
		dkgPartPubKey, _ := encoding.PointToStringHex(suite, keyPair.Public)
		gData.GenesisNode[validatorSeeds[i]] = &tpcmm.NodeInfo{
			NodeID:        validatorIDs[i],
			Weight:        10,
			DKGPartPubKey: dkgPartPubKey,
			Role:          tpcmm.NodeRole_Validator,
			State:         tpcmm.NodeState_Active,
		}
	}

	err := gData.Save("./genesis.json")
	assert.Equal(t, nil, err)
	gData.Load()
	assert.Equal(t, nil, err)
}
