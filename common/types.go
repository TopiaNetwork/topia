package common

type LedgerState byte

const (
	LedgerState_Uninitialized LedgerState = iota
	LedgerState_Genesis
	LedgerState_AutoInc
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

var CurrentNetworkType = NetworkType_Testnet

type NetworkType byte

const (
	NetworkType_Unknown NetworkType = iota
	NetworkType_Mainnet
	NetworkType_Testnet
)

func (n NetworkType) String() string {
	switch n {
	case NetworkType_Mainnet:
		return "Mainnet"
	case NetworkType_Testnet:
		return "Testnet"
	default:
		return "Unknown"
	}
}

func (n NetworkType) Value(netType byte) NetworkType {
	switch netType {
	case 'm':
		return NetworkType_Mainnet
	case 't':
		return NetworkType_Testnet
	default:
		return NetworkType_Unknown
	}
}

type NodeInfo struct {
	NodeID        string
	Weight        uint64
	DKGPartPubKey string
	Role          NodeRole
	State         NodeState
}

type EpochInfo struct {
	Epoch          uint64
	StartTimeStamp uint64
	StartHeight    uint64
}
