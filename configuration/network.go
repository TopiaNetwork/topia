package configuration

import (
	"time"

	tpcmm "github.com/TopiaNetwork/topia/common"
)

type PubSubConfiguration struct {
	ISSeedPeer            bool
	DirectPeers           []string
	IPColocationWhitelist []string
}

type SeedPeer struct {
	Role          tpcmm.NodeRole
	NetAddrString string
}

type ConnectionConfiguration struct {
	HighWater      int
	LowWater       int
	DurationPrune  time.Duration
	SeedPeers      []*SeedPeer
	ProtectedPeers []string //peer id string
}

type NetworkConfiguration struct {
	PubSub     *PubSubConfiguration
	Connection *ConnectionConfiguration
}

func DefPubSubConfiguration() *PubSubConfiguration {
	return &PubSubConfiguration{
		ISSeedPeer: false,
	}
}

func DefNetworkConfiguration() *NetworkConfiguration {
	return &NetworkConfiguration{
		PubSub:     DefPubSubConfiguration(),
		Connection: DefConnectionConfiguration(),
	}
}

func DefConnectionConfiguration() *ConnectionConfiguration {
	return &ConnectionConfiguration{
		HighWater:     50,
		LowWater:      200,
		DurationPrune: time.Second * 20,
		SeedPeers: []*SeedPeer{
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
		},
	}
}
