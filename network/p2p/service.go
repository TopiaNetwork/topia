package p2p

import (
	"context"
	"crypto/ecdsa"
	"fmt"
	"sync"
	"time"

	"github.com/btcsuite/btcd/btcec"
	"github.com/libp2p/go-libp2p"
	p2pCrypto "github.com/libp2p/go-libp2p-core/crypto"
	"github.com/libp2p/go-libp2p-core/host"
	"github.com/libp2p/go-libp2p-core/network"
	"github.com/libp2p/go-libp2p-core/peer"
	"github.com/libp2p/go-libp2p-core/protocol"
	dht "github.com/libp2p/go-libp2p-kad-dht"
	pubsub "github.com/libp2p/go-libp2p-pubsub"
	"github.com/multiformats/go-multiaddr"

	"github.com/AsynkronIT/protoactor-go/actor"

	"github.com/TopiaNetwork/topia/codec"
	tpcmm "github.com/TopiaNetwork/topia/common"
	tplog "github.com/TopiaNetwork/topia/log"
	logcomm "github.com/TopiaNetwork/topia/log/common"
	tpnetcmn "github.com/TopiaNetwork/topia/network/common"
	"github.com/TopiaNetwork/topia/network/message"
	tpnetprotoc "github.com/TopiaNetwork/topia/network/protocol"
)

type P2PService struct {
	sync.Mutex
	ctx           context.Context
	log           tplog.Logger
	host          host.Host
	pubsub        *pubsub.PubSub
	dht           *dht.IpfsDHT
	sysActor      *actor.ActorSystem
	modPIDS       map[string]*actor.PID      //module name -> actor PID
	modMarshals   map[string]codec.Marshaler //module name -> Marshaler
	dhtService    *P2PDHTService
	streamService *P2PStreamService
	pubsubService *P2PPubSubService
}

func NewP2PService(ctx context.Context, log tplog.Logger, sysActor *actor.ActorSystem, endPoint string, seed string) *P2PService {
	p2pLog := tplog.CreateModuleLogger(logcomm.InfoLevel, "P2PService", log)

	p2p := &P2PService{
		log:         p2pLog,
		sysActor:    sysActor,
		modPIDS:     make(map[string]*actor.PID),
		modMarshals: make(map[string]codec.Marshaler),
	}

	p2pPrivKey, err := p2p.createP2PPrivKey(seed)
	if err != nil {
		p2pLog.Errorf("createP2PPrivKey err: %s", seed)
		return nil
	}

	opts := []libp2p.Option{
		libp2p.ListenAddrStrings(endPoint),
		libp2p.Identity(p2pPrivKey),
		libp2p.DefaultMuxers,
		libp2p.DefaultSecurity,
		libp2p.DefaultTransports,
		libp2p.DisableRelay(),
		libp2p.NATPortMap(),
	}

	h, err := libp2p.New(opts...)
	if err != nil {
		p2pLog.Errorf("create p2p host err: %v", err)
		return nil
	}

	var bootNodes []string
	var dhtOptions []dht.Option
	if ctx.Value(tpnetcmn.NetContextKey_BOOTNODES) != nil {
		bootNodes = ctx.Value(tpnetcmn.NetContextKey_BOOTNODES).([]string)
	}
	if len(bootNodes) > 0 {
		dhtOptions = append(dhtOptions, p2p.withBootPeers(bootNodes))
	}
	dhtOptions = append(dhtOptions, p2p.defaultDHTOptions()...)
	dht, err := dht.New(ctx, h, dhtOptions...)
	if err != nil {
		p2pLog.Errorf("create p2p dht err: %v", err)
		return nil
	}

	err = dht.Bootstrap(ctx)
	if err != nil {
		p2pLog.Errorf("dht bootstrap err: %v", err)
		return nil
	}

	pubsub, err := pubsub.NewGossipSub(ctx, h, p2p.defaultPubSubOptions()...)
	if err != nil {
		p2pLog.Errorf("create p2p pubsub err: %v", err)
		return nil
	}

	p2p.ctx = ctx

	p2p.host = h
	p2p.dht = dht
	p2p.pubsub = pubsub

	p2p.dhtService = NewP2PDHTService(ctx, p2pLog, dht)
	p2p.streamService = NewP2PStreamService(ctx, p2pLog, p2p)
	p2p.pubsubService = NewP2PPubSubService(ctx, p2pLog, true, pubsub, p2p)

	p2p.host.SetStreamHandler(tpnetprotoc.AsyncSendProtocolID, p2p.streamService.handleIncomingStream)
	p2p.host.SetStreamHandler(tpnetprotoc.SyncProtocolID_Block, p2p.streamService.handleIncomingStreamWithResp)
	p2p.host.SetStreamHandler(tpnetprotoc.SyncProtocolID_Msg, p2p.streamService.handleIncomingStreamWithResp)
	p2p.host.SetStreamHandler(tpnetprotoc.HeatBeatPtotocolID, p2p.streamService.handleIncomingStreamWithResp)

	p2pLog.Info("create p2p service successfully")

	return p2p
}

func (p2p *P2PService) createP2PPrivKey(seed string) (*p2pCrypto.Secp256k1PrivateKey, error) {
	randReader, err := tpcmm.NewRandReader(seed)
	if err != nil {
		return nil, err
	}

	prvKey, _ := ecdsa.GenerateKey(btcec.S256(), randReader)

	return (*p2pCrypto.Secp256k1PrivateKey)(prvKey), nil
}

func (p2p *P2PService) defaultDHTOptions() []dht.Option {
	return []dht.Option{
		dht.ProtocolPrefix(tpnetprotoc.P2PProtocolPrefix),
		dht.Mode(dht.ModeServer),
	}
}

func (p2p *P2PService) defaultPubSubOptions() []pubsub.Option {
	return []pubsub.Option{
		pubsub.WithMaxMessageSize(tpnetprotoc.PubSubMaxMsgSize),
		pubsub.WithMessageSigning(true),
		pubsub.WithStrictSignatureVerification(true),
	}
}

func (p2p *P2PService) withBootPeers(bootPeers []string) dht.Option {
	var peers []peer.AddrInfo
	for _, b := range bootPeers {
		peerInfo, err := p2p.getAddrInfo(b)
		if err != nil {
			return nil
		}
		peers = append(peers, *peerInfo)
	}
	return dht.BootstrapPeers(peers...)
}

func (p2p *P2PService) ID() peer.ID {
	return p2p.host.ID()
}

func (p2p *P2PService) ListenAddr() []string {
	addrInfo := peer.AddrInfo{
		p2p.host.ID(),
		p2p.host.Addrs(),
	}

	maP2PAddrs, err := peer.AddrInfoToP2pAddrs(&addrInfo)
	if err != nil {
		p2p.log.Errorf("AddrInfoToP2pAddrs error: %v", err)
		return nil
	}

	var addrs []string
	for _, maAddr := range maP2PAddrs {
		addrs = append(addrs, maAddr.String())
	}

	return addrs
}

func (p2p *P2PService) RegisterModule(moduleName string, pid *actor.PID, msgMarshal codec.Marshaler) {
	p2p.Lock()
	defer p2p.Unlock()

	if _, ok := p2p.modPIDS[moduleName]; !ok {
		p2p.modPIDS[moduleName] = pid
	}

	if _, ok := p2p.modMarshals[moduleName]; !ok {
		p2p.modMarshals[moduleName] = msgMarshal
	}
}

func (p2p *P2PService) UnRegisterModule(moduleName string, pid *actor.PID, msgMarshal codec.Marshaler) {
	p2p.Lock()
	defer p2p.Unlock()

	delete(p2p.modPIDS, moduleName)
	delete(p2p.modMarshals, moduleName)
}

func (p2p *P2PService) dispatch(moduleName string, msg interface{}) error {
	if pid, ok := p2p.modPIDS[moduleName]; ok {
		//if msgMarshal, okM := p2p.modMarshals[moduleName]; okM {
		//	msg, err := msgMarshal.Unmarshal(data)
		//	if err == nil {
		p2p.sysActor.Root.Send(pid, msg)
		//	}
		//	p2p.log.Errorf("Unmarshal err(%s) module name =%s", err.Error(), moduleName)
		//	return err
		//} else {
		//	err := fmt.Errorf("can't find the module %s message marshals", moduleName)
		//	p2p.log.Error(err.Error())
		//	return err
		//}

		return nil
	} else {
		err := fmt.Errorf("can't find the module %s actor PID", moduleName)
		p2p.log.Error(err.Error())
		return err
	}
}

func (p2p *P2PService) dispatchAndWaitResp(moduleName string, streamMsg *message.NetworkMessage) (interface{}, error) {
	if pid, ok := p2p.modPIDS[moduleName]; ok {
		//if msgMarshal, okM := p2p.modMarshals[moduleName]; okM {
		//msg, err := msgMarshal.Unmarshal(data)
		//if err == nil {
		result, err := p2p.sysActor.Root.RequestFuture(pid, streamMsg.Data, tpnetprotoc.WaitRespTimeout).Result()
		if err != nil {
			p2p.log.Errorf("RequestFuture err(%s) module name =%s", err.Error(), moduleName)
			return nil, err
		} else {
			return result, nil
		}
		//}
		//p2p.log.Errorf("Unmarshal err(%s) module name =%s", err.Error(), moduleName)
		return nil, err
		//} else {
		//	err := fmt.Errorf("can't find the module %s message marshals", moduleName)
		//	p2p.log.Error(err.Error())
		//	return nil, err
		//}
	} else {
		err := fmt.Errorf("can't find the module %s actor PID", moduleName)
		p2p.log.Error(err.Error())
		return nil, err
	}
}

func (p2p *P2PService) Send(ctx context.Context, protocolID string, moduleName string, data []byte) error {
	var peerIDList []peer.ID
	if ctx.Value(tpnetcmn.NetContextKey_PeerList) != nil {
		peerList := ctx.Value(tpnetcmn.NetContextKey_PeerList).([]string)
		p2p.log.Debugf("SendWithResponse peerList: %v", peerList)
		if len(peerList) > 0 {
			for _, idStr := range peerList {
				id, _ := peer.Decode(idStr)
				peerIDList = append(peerIDList, id)
			}
		}
	}

	var rStrategy tpnetcmn.RouteStrategy
	if peerIDList == nil {
		if ctx.Value(tpnetcmn.NetContextKey_RouteStrategy) != nil {
			rStrategy = ctx.Value(tpnetcmn.NetContextKey_RouteStrategy).(tpnetcmn.RouteStrategy)
		}

		switch rStrategy {
		case tpnetcmn.RouteStrategy_Default:
			peerIDList, _ = p2p.dhtService.GetAllPeerIDs()
		case tpnetcmn.RouteStrategy_NearestBucket:
			peerIDList, _ = p2p.dhtService.GetNearestPeerIDs(p2p.host.ID())
		case tpnetcmn.RouteStrategy_BucketsWithFactor:
			peerIDList, _ = p2p.dhtService.GetPeersWithFactor()
		default:
			p2p.log.Debugf("Send with invalid RouteStrategy: %d", rStrategy)
		}
	}

	p2p.log.Debugf("Send peerIDList: %v with RouteStrategy %d", peerIDList, rStrategy)
	if len(peerIDList) == 0 {
		err := fmt.Errorf("There is no any target peer for SendWithResponse and check your network")
		p2p.log.Error(err.Error())
		return err
	}

	startTime := time.Now()

	var wg sync.WaitGroup
	for _, id := range peerIDList {
		wg.Add(1)
		go func(peerID peer.ID) {
			defer func() {
				wg.Done()
				singleDuration := time.Since(startTime)
				p2p.log.Debugf("Send time %d ms from peer %s to peer %s", singleDuration.Microseconds(), p2p.host.ID().String(), peerID.String())
			}()

			/*
				supported, err := p2p.host.Peerstore().SupportsProtocols(peerID, msg.ProtocolID)
				if err != nil {
					p2p.log.Errorf("failed to get protocols for peer: %w", err)
					return
				}

				if len(supported) == 0 || (supported[0] != msg.ProtocolID) {
					p2p.log.Errorf("peer %s does not support protocols %s: supported=%v",
						peerID, []string{msg.ProtocolID}, supported)

					return
				}
			*/

			stream, err := p2p.host.NewStream(
				network.WithNoDial(ctx, "should already have connection"),
				peerID,
				protocol.ID(protocolID),
			)
			if err != nil {
				p2p.log.Errorf("failed to open stream to peer %s: %v", peerID.String(), err)
				return
			}

			defer stream.Close() //nolint:errcheck

			msg := &message.NetworkMessage{
				FromPeerID: p2p.host.ID().String(),
				ProtocolID: protocolID,
				ModuleName: moduleName,
				Data:       data,
			}

			_ = stream.SetWriteDeadline(time.Now().Add(tpnetprotoc.WriteReqDeadline))
			if err := p2p.streamService.writeMessage(stream, msg); err != nil {
				_ = stream.SetWriteDeadline(time.Time{})
				p2p.log.Errorf("Stream sendMessage error: stream=%s, err=%s", stream.ID(), err.Error())
				return
			}
			_ = stream.SetWriteDeadline(time.Time{}) // clear deadline // FIXME: Needs
			//  its own API (https://github.com/libp2p/go-libp2p-core/issues/162).
		}(id)
	}
	wg.Wait()

	dur := time.Since(startTime)
	p2p.log.Infof("Finish sending request after %d ms", dur.Microseconds())

	return nil
}

func (p2p *P2PService) SendWithResponse(ctx context.Context, protocolID string, moduleName string, data []byte) ([][]byte, error) {
	var peerIDList []peer.ID
	if ctx.Value(tpnetcmn.NetContextKey_PeerList) != nil {
		peerList := ctx.Value(tpnetcmn.NetContextKey_PeerList).([]string)
		p2p.log.Debugf("SendWithResponse peerList: %v", peerList)
		if len(peerList) > 0 {
			for _, idStr := range peerList {
				id, _ := peer.Decode(idStr)
				peerIDList = append(peerIDList, id)
			}
		}
	}

	var rStrategy tpnetcmn.RouteStrategy
	if peerIDList == nil {
		if ctx.Value(tpnetcmn.NetContextKey_RouteStrategy) != nil {
			rStrategy = ctx.Value(tpnetcmn.NetContextKey_RouteStrategy).(tpnetcmn.RouteStrategy)
		}

		switch rStrategy {
		case tpnetcmn.RouteStrategy_Default:
			peerIDList, _ = p2p.dhtService.GetAllPeerIDs()
		case tpnetcmn.RouteStrategy_NearestBucket:
			peerIDList, _ = p2p.dhtService.GetNearestPeerIDs(p2p.host.ID())
		case tpnetcmn.RouteStrategy_BucketsWithFactor:
			peerIDList, _ = p2p.dhtService.GetPeersWithFactor()
		default:
			p2p.log.Debugf("SendWithResponse invalid with RouteStrategy: %d", rStrategy)
		}
	}

	p2p.log.Debugf("SendWithResponse peerIDList: %v with RouteStrategy %d", peerIDList, rStrategy)
	if len(peerIDList) == 0 {
		err := fmt.Errorf("There is no any target peer for SendWithResponse and check your network")
		p2p.log.Error(err.Error())
		return nil, err
	}

	startTime := time.Now()

	respCh := make(chan *message.NetworkMessage, len(peerIDList))
	var wg sync.WaitGroup
	for _, id := range peerIDList {
		wg.Add(1)
		go func(peerID peer.ID) {
			defer func() {
				wg.Done()
				singleDuration := time.Since(startTime)
				p2p.log.Debugf("Send time %d ms from peer %s to peer %s", singleDuration.Microseconds(), p2p.host.ID().String(), peerID.String())
			}()

			/*
				supported, err := p2p.host.Peerstore().SupportsProtocols(peerID, msg.ProtocolID)
				if err != nil {
					p2p.log.Errorf("failed to get protocols for peer: %w", err)
					return
				}
				if len(supported) == 0 || (supported[0] != msg.ProtocolID) {
					p2p.log.Errorf("peer %s does not support protocols %s",
						peerID, []string{msg.ProtocolID})

					return
				}
			*/

			stream, err := p2p.host.NewStream(
				network.WithNoDial(ctx, "should already have connection"),
				peerID,
				protocol.ID(protocolID))
			if err != nil {
				p2p.log.Errorf("failed to open stream to peer: %w", err)
				return
			}

			defer stream.Close() //nolint:errcheck

			msg := &message.NetworkMessage{
				FromPeerID: p2p.host.ID().String(),
				ProtocolID: protocolID,
				ModuleName: moduleName,
				Data:       data,
			}

			_ = stream.SetWriteDeadline(time.Now().Add(tpnetprotoc.WriteReqDeadline))
			if err := p2p.streamService.writeMessage(stream, msg); err != nil {
				_ = stream.SetWriteDeadline(time.Time{})
				p2p.log.Errorf("Stream sendMessage error: stream=%s, err=%s", stream.ID(), err.Error())
				return
			}
			_ = stream.SetWriteDeadline(time.Time{}) // clear deadline // FIXME: Needs
			//  its own API (https://github.com/libp2p/go-libp2p-core/issues/162).

			resp, err := p2p.streamService.readMessage(stream)
			if err == nil {
				respCh <- resp
			} else {
				p2p.log.Errorf("read resp error %v", err)
			}
		}(id)
	}
	wg.Wait()

	dur := time.Since(startTime)
	p2p.log.Debugf("received resp %d after %d ms", len(respCh), dur.Microseconds())

	threshold := 1
	if ctx.Value(tpnetcmn.NetContextKey_RespThreshold) != nil {
		thresholdR := ctx.Value(tpnetcmn.NetContextKey_RespThreshold).(float32)
		threshold = int(float32(len(peerIDList)) * thresholdR)
	}

	respLen := 0
	var respList [][]byte
	for r := range respCh {
		respLen++
		respList = append(respList, r.Data)
		if respLen >= len(respCh) || respLen >= threshold {
			break
		}
	}

	return respList, nil
}

func (p2p *P2PService) getAddrInfo(address string) (*peer.AddrInfo, error) {
	addrInfo, err := peer.AddrInfoFromString(address)
	if err != nil {
		err = fmt.Errorf("p2p: get peer node info error: address %s error %v", address, err)
		p2p.log.Error(err.Error())
		return nil, err
	}

	return addrInfo, nil
}

func (p2p *P2PService) Connect(listenAddr []string) error {
	var maP2PAddrs []multiaddr.Multiaddr
	for _, addr := range listenAddr {
		maAddr, err := multiaddr.NewMultiaddr(addr)
		if err != nil {
			p2p.log.Errorf("invalid listenAddr %s", addr)
			continue
		}
		maP2PAddrs = append(maP2PAddrs, maAddr)
	}

	addInfos, err := peer.AddrInfosFromP2pAddrs(maP2PAddrs...)
	if err != nil {
		p2p.log.Errorf("can't generate addrinfos: %v", err)
		return err
	}

	if len(addInfos) != 1 {
		p2p.log.Errorf("the listenAddr don't belong to the same peer: %s", listenAddr)
		return err
	}

	connRetryCnt := 0
	for connRetryCnt <= tpnetcmn.ConnectionRetry {
		connRetryCnt++
		err = p2p.host.Connect(p2p.ctx, addInfos[0])
		if err == nil {
			break
		}
	}

	if err != nil {
		p2p.log.Errorf("can't connect to %s after %d retry", listenAddr, connRetryCnt)
	}

	return err
}

func (p2p *P2PService) Subscribe(ctx context.Context, topic string, validators ...message.PubSubMessageValidator) error {
	return p2p.pubsubService.Subscribe(ctx, topic, validators...)
}

func (p2p *P2PService) UnSubscribe(topic string) error {
	return p2p.pubsubService.UnSubscribe(topic)
}

func (p2p *P2PService) Publish(ctx context.Context, topic string, data []byte) error {
	return p2p.pubsubService.Publish(ctx, topic, data)
}

func (p2p *P2PService) Start() {
}

func (p2p *P2PService) Close() {
	p2p.dht.Close()
	p2p.host.Close()
}
