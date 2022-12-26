package common

import (
	pubsub "github.com/libp2p/go-libp2p-pubsub"
	"github.com/libp2p/go-libp2p/core/network"
	"time"
)

type Connectedness network.Connectedness

const (
	NotConnected Connectedness = iota
	Connected
	CanConnect
	CannotConnect
)

func (cs Connectedness) String() string {
	return network.Connectedness(cs).String()
}

type Reachability network.Reachability

const (
	ReachabilityUnknown Reachability = iota
	ReachabilityPublic
	ReachabilityPrivate
)

func (rl Reachability) String() string {
	return network.Reachability(rl).String()
}

type PubsubScore struct {
	ID    string
	Score *pubsub.PeerScoreSnapshot
}

type NatInfo struct {
	Reachability Reachability
	PublicAddr   string
}

type RemotePeer struct {
	ID   string
	Addr string
}

type PeerDetail struct {
	ID                 string
	AgentVersion       string
	ConnectedAddrs     []string
	SupportedProtocols []string
	ConnInfos          *ConnInfos
}

type ConnInfos struct {
	FirstSeen time.Time
	Value     int
	Tags      map[string]int
	Conns     map[string]time.Time
}
