package service

import (
	"context"
	tpcmm "github.com/TopiaNetwork/topia/common"
	tpnet "github.com/TopiaNetwork/topia/network"
	tpnetcmn "github.com/TopiaNetwork/topia/network/common"
)

type NetworkService interface {
	ID() string

	NetworkType() tpcmm.NetworkType

	ListenAddr() []string

	ConnectedPeers() []*tpnetcmn.RemotePeer

	Connectedness(nodeID string) (tpnetcmn.Connectedness, error)

	PubSubScores() []tpnetcmn.PubsubScore

	NatState() (*tpnetcmn.NatInfo, error)

	PeerDetailInfo(nodeID string) (*tpnetcmn.PeerDetail, error)

	FindPeer(ctx context.Context, nodeID string) (string, error)

	ConnectToNode(ctx context.Context, nodeNetAddr string) error

	DisConnectWithNode(nodeID string) error
}

type networkService struct {
	tpnet.Network
}

func NewNetworkService(network tpnet.Network) NetworkService {
	return &networkService{network}
}

func (ns *networkService) NetworkType() tpcmm.NetworkType {
	return tpcmm.CurrentNetworkType
}
