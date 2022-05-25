package network

import (
	"context"
	"fmt"
	tpcmm "github.com/TopiaNetwork/topia/common"
	tpconfig "github.com/TopiaNetwork/topia/configuration"
	tplog "github.com/TopiaNetwork/topia/log"
	tplogcmm "github.com/TopiaNetwork/topia/log/common"
	tpnet "github.com/TopiaNetwork/topia/network"
	"github.com/stretchr/testify/assert"
	"testing"
	"time"
)

func TestNetworkSeedPeer(t *testing.T) {
	testLog, _ := tplog.CreateMainLogger(tplogcmm.InfoLevel, tplog.JSONFormat, tplog.StdErrOutput, "")

	netConfig := tpconfig.GetConfiguration().NetConfig

	netConfig.Connection.LowWater = 9

	netConfig.Connection.SeedPeers = []*tpconfig.SeedPeer{
		{tpcmm.NodeRole_Executor, "/ip4/127.0.0.1/tcp/41000/p2p/16Uiu2HAkvPb9xbeHsbDSS44xLY2ZWJvZtdUpcWBSAMuPBTx1eNnj"},
		{tpcmm.NodeRole_Executor, "/ip4/127.0.0.1/tcp/41001/p2p/16Uiu2HAm29UXXHcMbUXpeALDQuRppiRKxvCGnREN4XwKRTPgFV4Q"},
		{tpcmm.NodeRole_Executor, "/ip4/127.0.0.1/tcp/41002/p2p/16Uiu2HAm8r3jz6E9p1imUJh1wc4JprhHKDqBBZwu1gHsXSHJpM7d"},
		{tpcmm.NodeRole_Proposer, "/ip4/127.0.0.1/tcp/41003/p2p/16Uiu2HAmRGnTWGLtCJaH7VZKrrAPuh4waQueTyUGNMJ1giktViaP"},
		{tpcmm.NodeRole_Proposer, "/ip4/127.0.0.1/tcp/41004/p2p/16Uiu2HAmSxqKEEkjAHeUu2gpWUYdjEKbvz3t7S36haUtdkjPvgJu"},
		{tpcmm.NodeRole_Proposer, "/ip4/127.0.0.1/tcp/41005/p2p/16Uiu2HAkzgCXoLsa6g1iSjBY3Lykeqg2EF44ZJLBGykRpAaA8uY7"},
		{tpcmm.NodeRole_Validator, "/ip4/127.0.0.1/tcp/41006/p2p/16Uiu2HAmEjXsGN2yR6jXc1bPEiEs3o1aMu2apchx8taGArmKy7ja"},
		{tpcmm.NodeRole_Validator, "/ip4/127.0.0.1/tcp/41007/p2p/16Uiu2HAkxG2ZpVWub8xatogt7oP8cwu4pbbkxG1oQXrgNSembE5C"},
		{tpcmm.NodeRole_Validator, "/ip4/127.0.0.1/tcp/41008/p2p/16Uiu2HAmUpkpxGbEc4tUVPYmdaMeZS9VCbGiJV2mXFMiVCe71eDq"},
		{tpcmm.NodeRole_Validator, "/ip4/127.0.0.1/tcp/41009/p2p/16Uiu2HAmD57RgJxS7R4putawdjxz9KQqqtRnTDm4JoPeTjHTzN3R"},
	}

	var networks []tpnet.Network
	for i := 0; i < 10; i++ {
		seed := fmt.Sprintf("topia%d", i)
		port := fmt.Sprintf("4100%d", i)
		network := tpnet.NewNetwork(context.Background(), testLog, netConfig, nil, "/ip4/127.0.0.1/tcp/"+port, seed, NewNetworkActiveNodeMock())

		networks = append(networks, network)
	}

	for i := 0; i < 10; i++ {
		networks[i].Start()
	}

	time.Sleep(10 * time.Second)

	for i := 0; i < 10; i++ {
		connedPeer := networks[i].ConnectedPeers()
		assert.LessOrEqual(t, 9, len(connedPeer))
	}
}
