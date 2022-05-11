package configuration

type ChainConfiguration struct {
	MaxTargetItem uint64
	MaxCodeSize   uint64 //unit: byte
	BlockTxCount  uint64
}

func DefChainConfiguration() *ChainConfiguration {
	return &ChainConfiguration{
		MaxTargetItem: 50,
		MaxCodeSize:   10_000_000, //10MB
		BlockTxCount:  10000,
	}
}
