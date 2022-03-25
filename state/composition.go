package state

import (
	"crypto/sha256"
	"github.com/lazyledger/smt"
	"math/big"

	"github.com/TopiaNetwork/topia/chain"
	tpchaintypes "github.com/TopiaNetwork/topia/chain/types"
	tpcrtypes "github.com/TopiaNetwork/topia/crypt/types"
	"github.com/TopiaNetwork/topia/ledger"
	tplgss "github.com/TopiaNetwork/topia/ledger/state"
	tplog "github.com/TopiaNetwork/topia/log"
	tpnet "github.com/TopiaNetwork/topia/network"
	stateaccount "github.com/TopiaNetwork/topia/state/account"
	statechain "github.com/TopiaNetwork/topia/state/chain"
	statenode "github.com/TopiaNetwork/topia/state/node"
	staetround "github.com/TopiaNetwork/topia/state/round"
)

type NodeNetWorkStateWapper interface {
	GetActiveExecutorIDs() ([]string, error)

	GetActiveProposerIDs() ([]string, error)

	GetActiveValidatorIDs() ([]string, error)
}

type CompositionStateReadonly interface {
	GetNonce(addr tpcrtypes.Address) (uint64, error)

	GetBalance(symbol chain.TokenSymbol, addr tpcrtypes.Address) (*big.Int, error)

	ChainID() chain.ChainID

	NetworkType() tpnet.NetworkType

	GetLatestBlock() (*tpchaintypes.Block, error)

	GetAllConsensusNodes() ([]string, error)

	GetChainTotalWeight() (uint64, error)

	GetActiveExecutorIDs() ([]string, error)

	GetActiveProposerIDs() ([]string, error)

	GetActiveValidatorIDs() ([]string, error)

	GetNodeWeight(nodeID string) (uint64, error)

	GetActiveExecutorsTotalWeight() (uint64, error)

	GetActiveProposersTotalWeight() (uint64, error)

	GetActiveValidatorsTotalWeight() (uint64, error)

	GetCurrentRound() uint64

	SetCurrentRound(round uint64)

	GetCurrentEpoch() uint64

	SetCurrentEpoch(epoch uint64)

	StateRoot() ([]byte, error)

	StateLatestVersion() (uint64, error)

	StateVersions() ([]uint64, error)

	PendingStateStore() int32

	Stop() error

	Close() error
}

type CompositionState interface {
	stateaccount.AccountState
	statechain.ChainState
	statenode.NodeExecutorState
	statenode.NodeProposerState
	statenode.NodeValidatorState
	staetround.RoundState

	StateRoot() ([]byte, error)

	StateLatestVersion() (uint64, error)

	StateVersions() ([]uint64, error)

	PendingStateStore() int32

	Commit() error

	Rollback() error

	Stop() error

	Close() error
}

type compositionState struct {
	tplgss.StateStore
	stateaccount.AccountState
	statechain.ChainState
	statenode.NodeExecutorState
	statenode.NodeProposerState
	statenode.NodeValidatorState
	staetround.RoundState
	log    tplog.Logger
	ledger ledger.Ledger
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

func CreateCompositionState(log tplog.Logger, ledger ledger.Ledger) CompositionState {
	stateStore, _ := ledger.CreateStateStore()
	return &compositionState{
		log:                log,
		ledger:             ledger,
		StateStore:         stateStore,
		AccountState:       stateaccount.NewAccountState(stateStore),
		ChainState:         statechain.NewChainStore(stateStore),
		NodeExecutorState:  statenode.NewNodeExecutorState(stateStore),
		NodeProposerState:  statenode.NewNodeProposerState(stateStore),
		NodeValidatorState: statenode.NewNodeValidatorState(stateStore),
	}
}

func CreateCompositionStateReadonly(log tplog.Logger, ledger ledger.Ledger) CompositionState {
	stateStore, _ := ledger.CreateStateStoreReadonly()
	return &compositionState{
		log:                log,
		ledger:             ledger,
		StateStore:         stateStore,
		AccountState:       stateaccount.NewAccountState(stateStore),
		ChainState:         statechain.NewChainStore(stateStore),
		NodeExecutorState:  statenode.NewNodeExecutorState(stateStore),
		NodeProposerState:  statenode.NewNodeProposerState(stateStore),
		NodeValidatorState: statenode.NewNodeValidatorState(stateStore),
	}
}

func (cs *compositionState) PendingStateStore() int32 {
	return cs.ledger.PendingStateStore()
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

	nodeExecutorRoot, err := cs.GetNodeExecutorStateRoot()
	if err != nil {
		cs.log.Errorf("Can't get node executor state root: %v", err)
		return nil, err
	}

	nodeProposerRoot, err := cs.GetNodeProposerStateRoot()
	if err != nil {
		cs.log.Errorf("Can't get node proposer state root: %v", err)
		return nil, err
	}

	nodeValidatorRoot, err := cs.GetNodeValidatorStateRoot()
	if err != nil {
		cs.log.Errorf("Can't get node validator state root: %v", err)
		return nil, err
	}

	roundRoot, err := cs.GetRoundStateRoot()
	if err != nil {
		cs.log.Errorf("Can't get round state root: %v", err)
		return nil, err
	}

	tree := smt.NewSparseMerkleTree(smt.NewSimpleMap(), smt.NewSimpleMap(), sha256.New())
	tree.Update(accRoot, accRoot)
	tree.Update(chainRoot, chainRoot)
	tree.Update(nodeExecutorRoot, nodeExecutorRoot)
	tree.Update(nodeProposerRoot, nodeProposerRoot)
	tree.Update(nodeValidatorRoot, nodeValidatorRoot)
	tree.Update(roundRoot, roundRoot)

	return tree.Root(), nil
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