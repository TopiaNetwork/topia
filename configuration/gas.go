package configuration

type GasConfiguration struct {
	MinGasPrice     uint64
	MinGasLimit     uint64
	GasEachByte     uint64
	MaxGasEachBlock uint64
}
