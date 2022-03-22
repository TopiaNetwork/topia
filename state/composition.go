package state

import (
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
	"math/big"
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

	GetAllConsensusNodes() ([]string, error)

	GetChainTotalWeight() (uint64, error)

	GetNodeWeight(nodeID string) (uint64, error)

	GetLatestBlock() (*tpchaintypes.Block, error)

	GetActiveExecutorIDs() ([]string, error)

	GetActiveProposerIDs() ([]string, error)

	GetActiveValidatorIDs() ([]string, error)

	GetCurrentRound() uint64

	SetCurrentRound(round uint64)

	GetCurrentEpoch() uint64

	SetCurrentEpoch(epoch uint64)

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
