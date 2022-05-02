package universal

import "github.com/TopiaNetwork/topia/configuration"

func computeBasicGas(gasConfig *configuration.GasConfiguration, txUni *TransactionUniversalWithHead) uint64 {
	return gasConfig.MinGasLimit + txUni.DataLen()*gasConfig.GasEachByte
}
