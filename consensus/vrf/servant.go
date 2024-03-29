package vrf

import tpcmm "github.com/TopiaNetwork/topia/common"

type vrfServant interface {
	GetActiveExecutorIDs() []string

	GetActiveExecutorIDsOfDomain(domainID string) []string

	GetActiveValidatorIDs() []string

	GetActiveExecutorsTotalWeight() uint64

	GetActiveExecutorsTotalWeightOfDomain(domainID string) uint64

	GetActiveValidatorsTotalWeight() uint64

	GetNodeWeight(nodeID string) (uint64, error)

	GetLatestEpoch() *tpcmm.EpochInfo
}
