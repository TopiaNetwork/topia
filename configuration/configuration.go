package configuration

import (
	"os"
	"sync"
)

var config *Configuration
var once sync.Once

type Configuration struct {
	fsPath      string
	ChainConfig *ChainConfiguration
	NodeConfig  *NodeConfiguration
	CSConfig    *ConsensusConfiguration
	GasConfig   *GasConfiguration
	Genesis     *GenesisData
}

func GetConfiguration() *Configuration {
	once.Do(func() {
		genData := new(GenesisData)
		err := genData.Load("genesis.json")
		if err != nil {
			curDir, _ := os.Getwd()
			panic("Load genesis data err: " + err.Error() + ";Current Dir: " + curDir)
		}
		config = &Configuration{
			ChainConfig: DefChainConfiguration(),
			NodeConfig:  DefNodeConfiguration(),
			CSConfig:    DefConsensusConfiguration(),
			GasConfig:   DefGasConfiguration(),
			Genesis:     genData,
		}
	})

	return config
}
