package network

import (
	"context"

	"github.com/AsynkronIT/protoactor-go/actor"

	"github.com/TopiaNetwork/topia/codec"
	"github.com/TopiaNetwork/topia/configuration"
	tplog "github.com/TopiaNetwork/topia/log"
	tpnetcmn "github.com/TopiaNetwork/topia/network/common"
	"github.com/TopiaNetwork/topia/network/message"
	"github.com/TopiaNetwork/topia/network/p2p"
)

type Network interface {
	ID() string

	ListenAddr() []string

	Connect(listenAddr []string) error

	Send(ctx context.Context, protocolID string, moduleName string, data []byte) error

	SendWithResponse(ctx context.Context, protocolID string, moduleName string, data []byte) ([]message.SendResponse, error)

	Subscribe(ctx context.Context, topic string, localIgnore bool, validators ...message.PubSubMessageValidator) error

	UnSubscribe(topic string) error

	Publish(ctx context.Context, toModuleNames []string, topic string, data []byte) error

	RegisterModule(moduleName string, pid *actor.PID, marshaler codec.Marshaler)

	UnRegisterModule(moduleName string, pid *actor.PID, marshaler codec.Marshaler)

	UpdateNetActiveNode(netActiveNode tpnetcmn.NetworkActiveNode)

	ConnectedPeers() []*tpnetcmn.RemotePeer

	Connectedness(nodeID string) (tpnetcmn.Connectedness, error)

	PubSubScores() []tpnetcmn.PubsubScore

	NatState() (*tpnetcmn.NatInfo, error)

	PeerDetailInfo(nodeID string) (*tpnetcmn.PeerDetail, error)

	FindPeer(ctx context.Context, nodeID string) (string, error)

	ConnectToNode(ctx context.Context, nodeNetAddr string) error

	DisConnectWithNode(nodeID string) error

	Start()

	Ready() bool

	Stop()
}

type network struct {
	p2p *p2p.P2PService
}

func NewNetwork(ctx context.Context, log tplog.Logger, config *configuration.NetworkConfiguration, sysActor *actor.ActorSystem, endPoint string, seed string, netActiveNode tpnetcmn.NetworkActiveNode) Network {
	return &network{
		p2p: p2p.NewP2PService(ctx, log, config, sysActor, endPoint, seed, netActiveNode),
	}
}

func (net *network) ID() string {
	return net.p2p.ID().String()
}

func (net *network) ListenAddr() []string {
	return net.p2p.ListenAddr()
}

func (net *network) Connect(listenAddr []string) error {
	return net.p2p.Connect(listenAddr)
}

func (net *network) Send(ctx context.Context, protocolID string, moduleName string, data []byte) error {
	return net.p2p.Send(ctx, protocolID, moduleName, data)
}

func (net *network) SendWithResponse(ctx context.Context, protocolID string, moduleName string, data []byte) ([]message.SendResponse, error) {
	return net.p2p.SendWithResponse(ctx, protocolID, moduleName, data)
}

func (net *network) Subscribe(ctx context.Context, topic string, localIgnore bool, validators ...message.PubSubMessageValidator) error {
	return net.p2p.Subscribe(ctx, topic, localIgnore, validators...)
}

func (net *network) UnSubscribe(topic string) error {
	return net.p2p.UnSubscribe(topic)
}

func (net *network) Publish(ctx context.Context, toModuleNames []string, topic string, data []byte) error {
	return net.p2p.Publish(ctx, toModuleNames, topic, data)
}

func (net *network) RegisterModule(moduleName string, pid *actor.PID, marshaler codec.Marshaler) {
	net.p2p.RegisterModule(moduleName, pid, marshaler)
}

func (net *network) UnRegisterModule(moduleName string, pid *actor.PID, marshaler codec.Marshaler) {
	net.p2p.UnRegisterModule(moduleName, pid, marshaler)
}

func (net *network) UpdateNetActiveNode(netActiveNode tpnetcmn.NetworkActiveNode) {
	net.p2p.UpdateNetActiveNode(netActiveNode)
}

func (net *network) ConnectedPeers() []*tpnetcmn.RemotePeer {
	return net.p2p.ConnectedPeers()
}

func (net *network) Connectedness(nodeID string) (tpnetcmn.Connectedness, error) {
	return net.p2p.Connectedness(nodeID)
}

func (net *network) PubSubScores() []tpnetcmn.PubsubScore {
	return net.p2p.PubSubScores()
}

func (net *network) NatState() (*tpnetcmn.NatInfo, error) {
	return net.p2p.NatState()
}

func (net *network) PeerDetailInfo(nodeID string) (*tpnetcmn.PeerDetail, error) {
	return net.p2p.PeerDetailInfo(nodeID)
}

func (net *network) FindPeer(ctx context.Context, nodeID string) (string, error) {
	return net.p2p.FindPeer(ctx, nodeID)
}

func (net *network) ConnectToNode(ctx context.Context, nodeNetAddr string) error {
	return net.p2p.ConnectToNode(ctx, nodeNetAddr)
}

func (net *network) DisConnectWithNode(nodeID string) error {
	return net.p2p.DisConnectWithNode(nodeID)
}

func (net *network) Start() {
	net.p2p.Start()
}

func (net *network) Ready() bool {
	return net.p2p.Ready()
}

func (net *network) Stop() {
	net.p2p.Close()
}
