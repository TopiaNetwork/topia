package configuration

type ChainConfiguration struct {
	MaxTargetItem uint64
}

func DefChainConfiguration() *ChainConfiguration {
	return &ChainConfiguration{
		MaxTargetItem: 50,
	}
}
