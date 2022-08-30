package configuration

import (
	"os"
	"sync"
)

var config *Configuration
var once sync.Once

type Configuration struct {
	fsPath       string
	ChainConfig  *ChainConfiguration
	NodeConfig   *NodeConfiguration
	NetConfig    *NetworkConfiguration
	CSConfig     *ConsensusConfiguration
	TxPoolConfig *TransactionPoolConfig
	GasConfig    *GasConfiguration
	Genesis      *GenesisData
}

func GetConfiguration() *Configuration {
	once.Do(func() {
		genData := new(GenesisData)
		err := genData.Load()
		if err != nil {
			curDir, _ := os.Getwd()
			panic("Load genesis data err: " + err.Error() + ";Current Dir: " + curDir)
		}
		config = &Configuration{
			ChainConfig:  DefChainConfiguration(),
			NodeConfig:   DefNodeConfiguration(),
			NetConfig:    DefNetworkConfiguration(),
			CSConfig:     DefConsensusConfiguration(),
			TxPoolConfig: DefaultTransactionPoolConfig(),
			GasConfig:    DefGasConfiguration(),
			Genesis:      genData,
		}
	})

	return config
}
