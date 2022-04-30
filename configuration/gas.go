package configuration

type GasConfiguration struct {
	MinGasPrice      uint64
	GasPriceMultiple float64 //The multiple value of actual  gas price
	MinGasLimit      uint64
	GasEachByte      uint64
	MaxGasEachBlock  uint64
}

func DefGasConfiguration() *GasConfiguration {
	return &GasConfiguration{
		MinGasPrice:      1000000000,
		GasPriceMultiple: 1.0,
		MinGasLimit:      50000,
		GasEachByte:      1500,
		MaxGasEachBlock:  1500000000,
	}
}
