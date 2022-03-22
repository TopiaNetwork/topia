package configuration

import "sync"

var config *Configuration
var once sync.Once

type Configuration struct {
	fsPath      string
	ChainConfig *ChainConfig
	NodeConfig  *NodeConfiguration
	CSConfig    *ConsensusConfiguration
	GasConfig   *GasConfiguration
}

func GetConfiguration() *Configuration {
	once.Do(func() {
		config = &Configuration{}
	})

	return config
}
