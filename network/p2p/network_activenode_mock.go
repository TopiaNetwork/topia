package p2p

type NetworkActiveNodeMock struct {
	activeExcutors   []string
	activeProposers  []string
	activeValidators []string
}

func NewNetworkActiveNodeMock() *NetworkActiveNodeMock {
	return &NetworkActiveNodeMock{}
}

func (anmock *NetworkActiveNodeMock) addActiveExecutor(peerID string) {
	anmock.activeExcutors = append(anmock.activeExcutors, peerID)
}

func (anmock *NetworkActiveNodeMock) addActiveProposer(peerID string) {
	anmock.activeProposers = append(anmock.activeProposers, peerID)
}

func (anmock *NetworkActiveNodeMock) addActiveValidator(peerID string) {
	anmock.activeValidators = append(anmock.activeValidators, peerID)
}

func (anmock *NetworkActiveNodeMock) GetActiveExecutorIDs() ([]string, error) {
	return anmock.activeExcutors, nil
}

func (anmock *NetworkActiveNodeMock) GetActiveProposerIDs() ([]string, error) {
	return anmock.activeProposers, nil
}

func (anmock *NetworkActiveNodeMock) GetActiveValidatorIDs() ([]string, error) {
	return anmock.activeValidators, nil
}
