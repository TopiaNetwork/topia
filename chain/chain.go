package chain

type ChainID string

type TokenSymbol string

const (
	ChainID_Empty = ChainID("")
)

const (
	TokenSymbol_UnKnown TokenSymbol = ""
	TokenSymbol_Native              = "TPA"
)

const (
	CurrentCHainID ChainID = "topia_test"
)

type NodeState byte

const (
	NodeState_Unknown NodeState = iota
	NodeState_Active
	NodeState_Inactive
)

type NodeRole uint64

const (
	NodeRole_Unknown   NodeRole = 0x00
	NodeRole_Executor           = 0x01
	NodeRole_Proposer           = 0x02
	NodeRole_Validator          = 0x40
)

type NodeInfo struct {
	NodeID string
	Weight uint64
	Role   NodeRole
	State  NodeState
}
