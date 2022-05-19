package configuration

import "time"

type PubSubConfiguration struct {
	ISSeedPeer            bool
	DirectPeers           []string
	IPColocationWhitelist []string
}

type ConnectionConfiguration struct {
	HighWater      int
	LowWater       int
	DurationPrune  time.Duration
	SeedPeers      []string
	ProtectedPeers []string
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
	}
}
