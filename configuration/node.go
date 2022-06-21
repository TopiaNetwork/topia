package configuration

import "os"

type NodeConfiguration struct {
	RootPath string
}

func DefNodeConfiguration() *NodeConfiguration {
	homeDir, _ := os.UserHomeDir()
	return &NodeConfiguration{
		RootPath: homeDir,
	}
}
