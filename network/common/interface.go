package common

type NetworkActiveNode interface {
	GetActiveExecutorIDs() []string

	GetActiveProposerIDs() []string

	GetActiveValidatorIDs() []string
}
