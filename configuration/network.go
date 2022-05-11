package configuration

type PubSubConfiguration struct {
	Bootstrapper          bool
	DirectPeers           []string
	IPColocationWhitelist []string
}

type NetworkConfiguration struct {
	PubSub *PubSubConfiguration
}

func DefPubSubConfiguration() *PubSubConfiguration {
	return &PubSubConfiguration{
		Bootstrapper: false,
	}
}

func DefNetworkConfiguration() *NetworkConfiguration {
	return &NetworkConfiguration{
		PubSub: DefPubSubConfiguration(),
	}
}
