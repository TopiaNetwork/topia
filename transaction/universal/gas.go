package universal

import "github.com/TopiaNetwork/topia/configuration"

func computeBasicGas(gasConfig *configuration.GasConfiguration, txUniDataLen uint64) uint64 {
	return gasConfig.MinGasLimit + txUniDataLen*gasConfig.GasEachByte
}
