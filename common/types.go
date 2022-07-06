package common

const (
	EpochSpan = 300
)

type LedgerState byte

const (
	LedgerState_Uninitialized LedgerState = iota
	LedgerState_Genesis
	LedgerState_AutoInc
)

type DomainType byte

const (
	DomainType_Unknown DomainType = iota
	DomainType_Execute
	DomainType_Consensus
)

type NodeState byte

const (
	NodeState_Unknown NodeState = iota
	NodeState_Standby
	NodeState_Active
	NodeState_Frozen
)

type NodeRole uint64

const (
	NodeRole_Unknown   NodeRole = 0x00
	NodeRole_Executor           = 0x01
	NodeRole_Proposer           = 0x02
	NodeRole_Validator          = 0x04
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

func (n NodeRole) String() string {
	if n&NodeRole_Executor == NodeRole_Executor {
		return "executor"
	} else if n&NodeRole_Proposer == NodeRole_Proposer {
		return "proposer"
	} else if n&NodeRole_Validator == NodeRole_Validator {
		return "validator"
	} else {
		return "Unknown"
	}
}

func (n NodeRole) Value(role string) NodeRole {
	switch role {
	case "executor":
		return NodeRole_Executor
	case "proposer":
		return NodeRole_Proposer
	case "validator":
		return NodeRole_Validator
	default:
		return NodeRole_Unknown
	}
}

type NodeDomainMember struct {
	NodeID   string
	NodeRole NodeRole
	Weight   uint64
}

type NodeConsensusDomain struct {
	Threshold    int
	NParticipant int
	PublicKey    []byte
	PubShares    [][]byte
	Members      []*NodeDomainMember
}

type NodeExecuteDomain struct {
	Members []string //nodeIDs
}

type NodeDomainInfo struct {
	ID               string
	Type             DomainType
	ValidHeightStart uint64
	ValidHeightEnd   uint64
	CSDomainData     *NodeConsensusDomain
	ExeDomainData    *NodeExecuteDomain
}

type NodeInfo struct {
	NodeID        string
	Address       string
	Weight        uint64
	DKGPartPubKey string
	DKGPriShare   []byte
	Role          NodeRole
	State         NodeState
}

type EpochInfo struct {
	Epoch          uint64
	StartTimeStamp uint64
	StartHeight    uint64
}

func NodeIDs(nodeM []*NodeDomainMember) []string {
	var nIDs []string

	for _, nodeMember := range nodeM {
		nIDs = append(nIDs, nodeMember.NodeID)
	}

	return nIDs
}
