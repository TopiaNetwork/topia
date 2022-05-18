package configuration

type ChainConfiguration struct {
	MaxTargetItem         uint64
	MaxCodeSize           uint64 //unit: byte
	MaxTxCountOfEachBlock uint64
	MaxTxSizeOfEachBlock  uint64 /unit: byte
}

func DefChainConfiguration() *ChainConfiguration {
	return &ChainConfiguration{
		MaxTargetItem:         50,
		MaxCodeSize:           10_000_000, //10MB
		MaxTxCountOfEachBlock: 1000,
		MaxTxSizeOfEachBlock:  1000000,
	}
}
