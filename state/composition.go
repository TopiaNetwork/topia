package state

import (
	"crypto/sha256"
	"github.com/lazyledger/smt"
	"go.uber.org/atomic"
	"math/big"
	"sync"

	tpacc "github.com/TopiaNetwork/topia/account"
	tpchaintypes "github.com/TopiaNetwork/topia/chain/types"
	tpcmm "github.com/TopiaNetwork/topia/common"
	tpcrtypes "github.com/TopiaNetwork/topia/crypt/types"
	"github.com/TopiaNetwork/topia/currency"
	"github.com/TopiaNetwork/topia/ledger"
	tplgss "github.com/TopiaNetwork/topia/ledger/state"
	tplog "github.com/TopiaNetwork/topia/log"
	stateaccount "github.com/TopiaNetwork/topia/state/account"
	statechain "github.com/TopiaNetwork/topia/state/chain"
	statedomain "github.com/TopiaNetwork/topia/state/domain"
	stateepoch "github.com/TopiaNetwork/topia/state/epoch"
	statenode "github.com/TopiaNetwork/topia/state/node"
)

type NodeNetWorkStateWapper interface {
	GetActiveExecutorIDs() ([]string, error)

	GetActiveProposerIDs() ([]string, error)

	GetActiveValidatorIDs() ([]string, error)
}

type CompositionStateReadonly interface {
	ChainID() tpchaintypes.ChainID

	NetworkType() tpcmm.NetworkType

	GetAccountRoot() ([]byte, error)

	GetAccountProof(addr tpcrtypes.Address) ([]byte, error)

	IsAccountExist(addr tpcrtypes.Address) bool

	GetAccount(addr tpcrtypes.Address) (*tpacc.Account, error)

	GetNonce(addr tpcrtypes.Address) (uint64, error)

	GetBalance(addr tpcrtypes.Address, symbol currency.TokenSymbol) (*big.Int, error)

	GetAllAccounts() ([]*tpacc.Account, error)

	GetChainRoot() ([]byte, error)

	GetLatestBlock() (*tpchaintypes.Block, error)

	GetLatestBlockResult() (*tpchaintypes.BlockResult, error)

	GetNodeDomainRoot() ([]byte, error)

	GetNodeExecuteDomainRoot() ([]byte, error)

	GetNodeConsensusDomainRoot() ([]byte, error)

	IsNodeDomainExist(nodeID string) bool

	IsExistNodeExecuteDomain(id string) bool

	IsExistNodeConsensusDomain(id string) bool

	GetNodeDomain(id string) (*tpcmm.NodeDomainInfo, error)

	GetAllActiveNodeDomains(height uint64) ([]*tpcmm.NodeDomainInfo, error)

	GetAllActiveNodeExecuteDomains(height uint64) ([]*tpcmm.NodeDomainInfo, error)

	GetAllActiveNodeConsensusDomains(height uint64) ([]*tpcmm.NodeDomainInfo, error)

	GetNodeRoot() ([]byte, error)

	GetNodeExecutorRoot() ([]byte, error)

	GetNodeProposerRoot() ([]byte, error)

	GetNodeValidatorRoot() ([]byte, error)

	GetNodeInactiveRoot() ([]byte, error)

	IsNodeExist(nodeID string) bool

	IsExistActiveExecutor(nodeID string) bool

	IsExistActiveProposer(nodeID string) bool

	IsExistActiveValidator(nodeID string) bool

	IsExistInactiveNode(nodeID string) bool

	GetAllConsensusNodeIDs() ([]string, error)

	GetNode(nodeID string) (*tpcmm.NodeInfo, error)

	GetTotalWeight() (uint64, error)

	GetActiveExecutorIDs() ([]string, error)

	GetActiveProposerIDs() ([]string, error)

	GetActiveValidatorIDs() ([]string, error)

	GetInactiveNodeIDs() ([]string, error)

	GetAllActiveExecutors() ([]*tpcmm.NodeInfo, error)

	GetAllActiveProposers() ([]*tpcmm.NodeInfo, error)

	GetAllActiveValidators() ([]*tpcmm.NodeInfo, error)

	GetAllInactiveNodes() ([]*tpcmm.NodeInfo, error)

	GetNodeWeight(nodeID string) (uint64, error)

	GetDKGPartPubKeysForVerify() (map[string]string, error)

	GetActiveExecutorsTotalWeight() (uint64, error)

	GetActiveProposersTotalWeight() (uint64, error)

	GetActiveValidatorsTotalWeight() (uint64, error)

	GetEpochRoot() ([]byte, error)

	GetLatestEpoch() (*tpcmm.EpochInfo, error)

	CompSStateStore() tplgss.StateStore

	StateRoot() ([]byte, error)

	StateLatestVersion() (uint64, error)

	StateVersions() ([]uint64, error)

	PendingStateStore() int32

	SnapToMem(log tplog.Logger) CompositionState

	Stop() error

	Close() error
}

type CompSState uint32

const (
	CompSState_Unknown CompSState = iota
	CompSState_Idle
	CompSState_Executed
	CompSState_Commited
)

type TakenUpState uint32

const (
	TakenUpState_Unknown TakenUpState = iota
	TakenUpState_Idle
	TakenUpState_Busy
)

type CompositionState interface {
	stateaccount.AccountState
	statechain.ChainState
	statedomain.NodeDomainState
	statedomain.NodeExecuteDomainState
	statedomain.NodeConsensusDomainState
	statenode.NodeState
	statenode.NodeInactiveState
	statenode.NodeExecutorState
	statenode.NodeProposerState
	statenode.NodeValidatorState
	stateepoch.EpochState

	CompSStateStore() tplgss.StateStore

	StateVersion() uint64

	CompSState() CompSState

	StateRoot() ([]byte, error)

	StateLatestVersion() (uint64, error)

	StateVersions() ([]uint64, error)

	PendingStateStore() int32

	UpdateCompSState(state CompSState)

	SnapToMem(log tplog.Logger) CompositionState

	Lock()

	Unlock()

	Commit() error

	Rollback() error

	Stop() error

	Close() error
}

type compositionState struct {
	tplgss.StateStore
	stateaccount.AccountState
	statechain.ChainState
	statedomain.NodeDomainState
	statedomain.NodeExecuteDomainState
	statedomain.NodeConsensusDomainState
	statenode.NodeState
	statenode.NodeInactiveState
	statenode.NodeExecutorState
	statenode.NodeProposerState
	statenode.NodeValidatorState
	stateepoch.EpochState
	log          tplog.Logger
	ledger       ledger.Ledger
	stateVersion uint64
	state        atomic.Uint32 //CompSState
	sync         sync.RWMutex
}

type nodeNetWorkStateWapper struct {
	log    tplog.Logger
	ledger ledger.Ledger
}

func NewNodeNetWorkStateWapper(log tplog.Logger, ledger ledger.Ledger) NodeNetWorkStateWapper {
	return &nodeNetWorkStateWapper{
		log:    log,
		ledger: ledger,
	}
}

func createCompositionStateWithStateStore(log tplog.Logger, ledger ledger.Ledger, stateVersion uint64, stateStore tplgss.StateStore) *compositionState {
	exeNodeDomainState := statedomain.NewNodeExecuteDomainState(stateStore, 1)
	csNodeDomainState := statedomain.NewNodeConsensusDomainState(stateStore, 1)
	nodeDomainState := statedomain.NewNodeDomainState(stateStore, exeNodeDomainState, csNodeDomainState, 1)
	inactiveState := statenode.NewNodeInactiveState(stateStore, 1) // 1 Megabyte
	executorState := statenode.NewNodeExecutorState(stateStore, 1)
	proposerState := statenode.NewNodeProposerState(stateStore, 1)
	validatorState := statenode.NewNodeValidatorState(stateStore, 1)
	nodeState := statenode.NewNodeState(stateStore, inactiveState, executorState, proposerState, validatorState, 1)
	return &compositionState{
		log:                      log,
		stateVersion:             stateVersion,
		ledger:                   ledger,
		StateStore:               stateStore,
		AccountState:             stateaccount.NewAccountState(stateStore, 1),
		ChainState:               statechain.NewChainStore(ledger.ID(), stateStore, ledger, 1),
		NodeDomainState:          nodeDomainState,
		NodeExecuteDomainState:   exeNodeDomainState,
		NodeConsensusDomainState: csNodeDomainState,
		NodeState:                nodeState,
		NodeInactiveState:        inactiveState,
		NodeExecutorState:        executorState,
		NodeProposerState:        proposerState,
		NodeValidatorState:       validatorState,
		EpochState:               stateepoch.NewEpochState(ledger.ID(), stateStore, 1),
	}
}

func createCompositionState(log tplog.Logger, ledger ledger.Ledger, stateVersion uint64) CompositionState {
	stateStore, _ := ledger.CreateStateStore()

	compS := createCompositionStateWithStateStore(log, ledger, stateVersion, stateStore)
	compS.state.Store(uint32(CompSState_Idle))

	return compS
}

func CreateCompositionStateReadonly(log tplog.Logger, ledger ledger.Ledger) CompositionStateReadonly {
	stateStore, _ := ledger.CreateStateStoreReadonly()

	return createCompositionStateWithStateStore(log, ledger, 0, stateStore)
}

func CreateCompositionStateReadonlyAt(log tplog.Logger, ledger ledger.Ledger, version uint64) CompositionStateReadonly {
	stateStore, _ := ledger.CreateStateStoreReadonlyAt(version)

	return createCompositionStateWithStateStore(log, ledger, 0, stateStore)
}

func CreateCompositionStateMem(log tplog.Logger, compState CompositionStateReadonly) CompositionState {
	stateStore, ledger, _ := ledger.CreateStateStoreMem(log)

	compS := createCompositionStateWithStateStore(log, ledger, 1, stateStore)

	compS.state.Store(uint32(CompSState_Idle))

	originStateStore := compState.CompSStateStore()
	err := originStateStore.Clone(stateStore)
	if err != nil {
		log.Errorf("Can't clone orgin keys and values: %v", err)
		return nil
	}

	return compS
}

func (cs *compositionState) PendingStateStore() int32 {
	return cs.ledger.PendingStateStore()
}

func (cs *compositionState) CompSStateStore() tplgss.StateStore {
	return cs.StateStore
}

func (cs *compositionState) StateRoot() ([]byte, error) {
	accRoot, err := cs.GetAccountRoot()
	if err != nil {
		cs.log.Errorf("Can't get account state root: %v", err)
		return nil, err
	}

	chainRoot, err := cs.GetChainRoot()
	if err != nil {
		cs.log.Errorf("Can't get chain state root: %v", err)
		return nil, err
	}

	domainRoot, err := cs.GetNodeDomainRoot()
	if err != nil {
		cs.log.Errorf("Can't get node domain state root: %v", err)
		return nil, err
	}

	exeDomainRoot, err := cs.GetNodeExecuteDomainRoot()
	if err != nil {
		cs.log.Errorf("Can't get node execute domain state root: %v", err)
		return nil, err
	}

	csDomainRoot, err := cs.GetNodeConsensusDomainRoot()
	if err != nil {
		cs.log.Errorf("Can't get node consensus domain state root: %v", err)
		return nil, err
	}

	inactiveRoot, err := cs.GetNodeInactiveRoot()
	if err != nil {
		cs.log.Errorf("Can't get inactive node state root: %v", err)
		return nil, err
	}

	nodeRoot, err := cs.GetNodeRoot()
	if err != nil {
		cs.log.Errorf("Can't get node state root: %v", err)
		return nil, err
	}

	nodeExecutorRoot, err := cs.GetNodeExecutorRoot()
	if err != nil {
		cs.log.Errorf("Can't get node executor state root: %v", err)
		return nil, err
	}

	nodeProposerRoot, err := cs.GetNodeProposerRoot()
	if err != nil {
		cs.log.Errorf("Can't get node proposer state root: %v", err)
		return nil, err
	}

	nodeValidatorRoot, err := cs.GetNodeValidatorRoot()
	if err != nil {
		cs.log.Errorf("Can't get node validator state root: %v", err)
		return nil, err
	}

	epochRoot, err := cs.GetEpochRoot()
	if err != nil {
		cs.log.Errorf("Can't get epoch state root: %v", err)
		return nil, err
	}

	tree := smt.NewSparseMerkleTree(smt.NewSimpleMap(), smt.NewSimpleMap(), sha256.New())
	tree.Update(accRoot, accRoot)
	tree.Update(chainRoot, chainRoot)
	tree.Update(domainRoot, domainRoot)
	tree.Update(exeDomainRoot, exeDomainRoot)
	tree.Update(csDomainRoot, csDomainRoot)
	tree.Update(inactiveRoot, inactiveRoot)
	tree.Update(nodeRoot, nodeRoot)
	tree.Update(nodeExecutorRoot, nodeExecutorRoot)
	tree.Update(nodeProposerRoot, nodeProposerRoot)
	tree.Update(nodeValidatorRoot, nodeValidatorRoot)
	tree.Update(epochRoot, epochRoot)

	return tree.Root(), nil
}

func (cs *compositionState) StateVersion() uint64 {
	return cs.stateVersion
}

func (cs *compositionState) CompSState() CompSState {
	return CompSState(cs.state.Load())
}

func (cs *compositionState) UpdateCompSState(state CompSState) {
	cs.state.Swap(uint32(state))
}

func (cs *compositionState) SnapToMem(log tplog.Logger) CompositionState {
	return CreateCompositionStateMem(log, cs)
}

func (cs *compositionState) Lock() {
	cs.sync.Lock()
}

func (cs *compositionState) Unlock() {
	cs.sync.Unlock()
}

func (nw *nodeNetWorkStateWapper) GetActiveExecutorIDs() ([]string, error) {
	csStateRN := CreateCompositionStateReadonly(nw.log, nw.ledger)
	defer csStateRN.Stop()

	return csStateRN.GetActiveExecutorIDs()
}

func (nw *nodeNetWorkStateWapper) GetActiveProposerIDs() ([]string, error) {
	csStateRN := CreateCompositionStateReadonly(nw.log, nw.ledger)
	defer csStateRN.Stop()

	return csStateRN.GetActiveProposerIDs()
}

func (nw *nodeNetWorkStateWapper) GetActiveValidatorIDs() ([]string, error) {
	csStateRN := CreateCompositionStateReadonly(nw.log, nw.ledger)
	defer csStateRN.Stop()

	return csStateRN.GetActiveValidatorIDs()
}

func GetLatestBlock(ledger ledger.Ledger) (*tpchaintypes.Block, error) {
	b, err := statechain.GetLatestBlock(ledger.ID())
	if err != nil {
		stateStore, _ := ledger.CreateStateStoreReadonly()
		cState := statechain.NewChainStore(ledger.ID(), stateStore, ledger, 1)
		b, err = cState.GetLatestBlock()
	}

	return b, err
}

func GetLatestBlockResult(ledger ledger.Ledger) (*tpchaintypes.BlockResult, error) {
	brs, err := statechain.GetLatestBlockResult(ledger.ID())
	if err != nil {
		stateStore, _ := ledger.CreateStateStoreReadonly()
		cState := statechain.NewChainStore(ledger.ID(), stateStore, ledger, 1)
		brs, err = cState.GetLatestBlockResult()
	}

	return brs, err
}

func GetLatestEpoch(ledger ledger.Ledger) (*tpcmm.EpochInfo, error) {
	brs, err := stateepoch.GetLatestEpoch(ledger.ID())
	if err != nil {
		stateStore, _ := ledger.CreateStateStoreReadonly()
		cState := stateepoch.NewEpochState(ledger.ID(), stateStore, 1)
		brs, err = cState.GetLatestEpoch()
	}

	return brs, err
}
