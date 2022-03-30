package network

import (
	"context"

	"github.com/AsynkronIT/protoactor-go/actor"

	"github.com/TopiaNetwork/topia/codec"
	tplog "github.com/TopiaNetwork/topia/log"
	tpnetcmn "github.com/TopiaNetwork/topia/network/common"
	"github.com/TopiaNetwork/topia/network/message"
	"github.com/TopiaNetwork/topia/network/p2p"
)

var CurrentNetworkType = NetworkType_Testnet

type NetworkType byte

const (
	NetworkType_Unknown NetworkType = iota
	NetworkType_Mainnet
	NetworkType_Testnet
)

type Network interface {
	ID() string

	ListenAddr() []string

	Connect(listenAddr []string) error

	Send(ctx context.Context, protocolID string, moduleName string, data []byte) error

	SendWithResponse(ctx context.Context, protocolID string, moduleName string, data []byte) ([][]byte, error)

	Subscribe(ctx context.Context, topic string, validators ...message.PubSubMessageValidator) error

	UnSubscribe(topic string) error

	Publish(ctx context.Context, toModuleName string, topic string, data []byte) error

	RegisterModule(moduleName string, pid *actor.PID, marshaler codec.Marshaler)

	UnRegisterModule(moduleName string, pid *actor.PID, marshaler codec.Marshaler)

	Start()

	Stop()
}

type network struct {
	p2p *p2p.P2PService
}

func NewNetwork(ctx context.Context, log tplog.Logger, sysActor *actor.ActorSystem, endPoint string, seed string, netActiveNode tpnetcmn.NetworkActiveNode) Network {
	return &network{
		p2p: p2p.NewP2PService(ctx, log, sysActor, endPoint, seed, netActiveNode),
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

func (net *network) SendWithResponse(ctx context.Context, protocolID string, moduleName string, data []byte) ([][]byte, error) {
	return net.p2p.SendWithResponse(ctx, protocolID, moduleName, data)
}

func (net *network) Subscribe(ctx context.Context, topic string, validators ...message.PubSubMessageValidator) error {
	return net.p2p.Subscribe(ctx, topic, validators...)
}

func (net *network) UnSubscribe(topic string) error {
	return net.p2p.UnSubscribe(topic)
}

func (net *network) Publish(ctx context.Context, toModuleName string, topic string, data []byte) error {
	return net.p2p.Publish(ctx, toModuleName, topic, data)
}

func (net *network) RegisterModule(moduleName string, pid *actor.PID, marshaler codec.Marshaler) {
	net.p2p.RegisterModule(moduleName, pid, marshaler)
}

func (net *network) UnRegisterModule(moduleName string, pid *actor.PID, marshaler codec.Marshaler) {
	net.p2p.UnRegisterModule(moduleName, pid, marshaler)
}

func (net *network) Start() {
	net.p2p.Start()
}

func (net *network) Stop() {
	net.p2p.Close()
}

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
