package configuration

import "sync"

var config *Configuration
var once sync.Once

type Configuration struct {
	fsPath      string
	ChainConfig *ChainConfiguration
	NodeConfig  *NodeConfiguration
	CSConfig    *ConsensusConfiguration
	GasConfig   *GasConfiguration
}

func GetConfiguration() *Configuration {
	once.Do(func() {
		config = &Configuration{
			ChainConfig: DefChainConfiguration(),
			NodeConfig:  DefNodeConfiguration(),
			CSConfig:    DefConsensusConfiguration(),
			GasConfig:   DefGasConfiguration(),
		}
	})

	return config
}
