package configuration

type GasConfiguration struct {
	MinGasPrice     uint64
	MinGasLimit     uint64
	GasEachByte     uint64
	MaxGasEachBlock uint64
}

func DefGasConfiguration() *GasConfiguration {
	return &GasConfiguration{
		MinGasPrice:     1000000000,
		MinGasLimit:     50000,
		GasEachByte:     1500,
		MaxGasEachBlock: 1500000000,
	}
}
