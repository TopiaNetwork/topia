package p2p

import (
	"context"
	"crypto/ecdsa"
	"fmt"
	"net"
	"sort"
	"strings"
	"sync"
	"time"

	"github.com/AsynkronIT/protoactor-go/actor"
	"github.com/btcsuite/btcd/btcec"
	"github.com/libp2p/go-libp2p"
	"github.com/libp2p/go-libp2p-connmgr"
	p2pCrypto "github.com/libp2p/go-libp2p-core/crypto"
	"github.com/libp2p/go-libp2p-core/host"
	"github.com/libp2p/go-libp2p-core/network"
	"github.com/libp2p/go-libp2p-core/peer"
	"github.com/libp2p/go-libp2p-core/protocol"
	dht "github.com/libp2p/go-libp2p-kad-dht"
	pubsub "github.com/libp2p/go-libp2p-pubsub"
	swarm "github.com/libp2p/go-libp2p-swarm"
	basichost "github.com/libp2p/go-libp2p/p2p/host/basic"
	ma "github.com/multiformats/go-multiaddr"

	"github.com/TopiaNetwork/topia/codec"
	tpcmm "github.com/TopiaNetwork/topia/common"
	"github.com/TopiaNetwork/topia/configuration"
	tplog "github.com/TopiaNetwork/topia/log"
	logcomm "github.com/TopiaNetwork/topia/log/common"
	tpnetcmn "github.com/TopiaNetwork/topia/network/common"
	"github.com/TopiaNetwork/topia/network/message"
	tpnetprotoc "github.com/TopiaNetwork/topia/network/protocol"
	"github.com/TopiaNetwork/topia/version"
)

type DHTServiceType int

const (
	DHTServiceType_Unknown = iota
	DHTServiceType_General
	DHTServiceType_Execute
	DHTServiceType_Propose
	DHTServiceType_Validate
)

const (
	GossipScoreThreshold             = -500
	PublishScoreThreshold            = -1000
	GraylistScoreThreshold           = -2500
	AcceptPXScoreThreshold           = 1000
	OpportunisticGraftScoreThreshold = 3.5
)

func init() {
	// configure larger overlay parameters
	pubsub.GossipSubD = 8
	pubsub.GossipSubDscore = 6
	pubsub.GossipSubDout = 3
	pubsub.GossipSubDlo = 6
	pubsub.GossipSubDhi = 12
	pubsub.GossipSubDlazy = 12
	pubsub.GossipSubDirectConnectInitialDelay = 30 * time.Second
	pubsub.GossipSubIWantFollowupTime = 5 * time.Second
	pubsub.GossipSubHistoryLength = 10
	pubsub.GossipSubGossipFactor = 0.1
}

type P2PService struct {
	sync.Mutex
	ctx            context.Context
	log            tplog.Logger
	config         *configuration.NetworkConfiguration
	host           host.Host
	pubsub         *pubsub.PubSub
	sysActor       *actor.ActorSystem
	modPIDS        map[string]*actor.PID      //module name -> actor PID
	modMarshals    map[string]codec.Marshaler //module name -> Marshaler
	netActiveNode  tpnetcmn.NetworkActiveNode
	dhtServices    map[DHTServiceType]*P2PDHTService
	streamService  *P2PStreamService
	pubsubService  *P2PPubSubService
	peerScoreCache *PeerScoreCache
}

func NewP2PService(ctx context.Context, log tplog.Logger, config *configuration.NetworkConfiguration, sysActor *actor.ActorSystem, endPoint string, seed string, netActiveNode tpnetcmn.NetworkActiveNode) *P2PService {
	p2pLog := tplog.CreateModuleLogger(logcomm.InfoLevel, "P2PService", log)

	p2p := &P2PService{
		ctx:            ctx,
		log:            p2pLog,
		config:         config,
		sysActor:       sysActor,
		modPIDS:        make(map[string]*actor.PID),
		modMarshals:    make(map[string]codec.Marshaler),
		netActiveNode:  netActiveNode,
		peerScoreCache: NewPeerScoreCache(),
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
		libp2p.UserAgent("topia-" + version.TMVersion),
		p2p.connectionManagerOption(),
	}

	h, err := libp2p.New(opts...)
	if err != nil {
		p2pLog.Errorf("create p2p host err: %v", err)
		return nil
	}

	pubsub, err := pubsub.NewGossipSub(ctx, h, p2p.defaultPubSubOptions()...)
	if err != nil {
		p2pLog.Errorf("create p2p pubsub err: %v", err)
		return nil
	}

	p2p.ctx = ctx

	p2p.host = h
	p2p.pubsub = pubsub

	if err = p2p.createDHTService(ctx, p2pLog, h); err != nil {
		p2pLog.Errorf("create DHT service err: %v", err)
		return nil
	}

	p2p.streamService = NewP2PStreamService(ctx, p2pLog, p2p)
	p2p.pubsubService = NewP2PPubSubService(ctx, p2pLog, true, pubsub, p2p)

	p2p.host.SetStreamHandler(tpnetprotoc.AsyncSendProtocolID, p2p.streamService.handleIncomingStream)
	p2p.host.SetStreamHandler(tpnetprotoc.SyncProtocolID_Block, p2p.streamService.handleIncomingStreamWithResp)
	p2p.host.SetStreamHandler(tpnetprotoc.SyncProtocolID_Msg, p2p.streamService.handleIncomingStreamWithResp)
	p2p.host.SetStreamHandler(tpnetprotoc.HeatBeatPtotocolID, p2p.streamService.handleIncomingStreamWithResp)
	p2p.host.SetStreamHandler(tpnetprotoc.ForwardExecute_Msg, p2p.streamService.handleIncomingStream)
	p2p.host.SetStreamHandler(tpnetprotoc.ForwardExecute_SyncMsg, p2p.streamService.handleIncomingStreamWithResp)
	p2p.host.SetStreamHandler(tpnetprotoc.ForwardPropose_Msg, p2p.streamService.handleIncomingStream)
	p2p.host.SetStreamHandler(tpnetprotoc.FrowardValidate_Msg, p2p.streamService.handleIncomingStream)

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

func (p2p *P2PService) connectionManagerOption() libp2p.Option {
	connMng, err := connmgr.NewConnManager(p2p.config.Connection.LowWater, p2p.config.Connection.HighWater, connmgr.WithGracePeriod(p2p.config.Connection.DurationPrune))
	if err != nil {
		return nil
	}

	for _, protPeer := range p2p.config.Connection.ProtectedPeers {
		peerID, err := peer.IDFromString(protPeer)
		if err != nil {
			panic(fmt.Sprintf("failed to parse peer ID in protected peers array: %v", err))
			return nil
		}

		connMng.Protect(peerID, "config-prot")
	}

	for _, seedPeer := range p2p.config.Connection.SeedPeers {
		peerID, err := peer.IDFromString(seedPeer.NetAddrString)
		if err != nil {
			panic(fmt.Sprintf("failed to parse peer ID in boot peers array: %v", err))
			return nil
		}
		connMng.Protect(peerID, "seedpeer")
	}

	return libp2p.ConnectionManager(connMng)
}

func (p2p *P2PService) defaultDHTOptions() []dht.Option {
	return []dht.Option{
		dht.ProtocolPrefix(tpnetprotoc.P2PProtocolPrefix),
		dht.Mode(dht.ModeServer),
	}
}

func (p2p *P2PService) defaultDHTOptionsExecute() []dht.Option {
	return []dht.Option{
		dht.ProtocolPrefix(tpnetprotoc.P2PProtocolExecutePrefix),
		dht.Mode(dht.ModeServer),
	}
}

func (p2p *P2PService) defaultDHTOptionsPropose() []dht.Option {
	return []dht.Option{
		dht.ProtocolPrefix(tpnetprotoc.P2PProtocolProposePrefix),
		dht.Mode(dht.ModeServer),
	}
}

func (p2p *P2PService) defaultDHTOptionsValidate() []dht.Option {
	return []dht.Option{
		dht.ProtocolPrefix(tpnetprotoc.P2PProtocolValidatePrefix),
		dht.Mode(dht.ModeServer),
	}
}

func (p2p *P2PService) isBootNode(p peer.ID) bool {
	bootNodes, bootNodesExecute, bootNodesPropose, bootNodesValidate := p2p.getBootNodes(p2p.ctx)

	bootNodesAll := append(append(bootNodes, bootNodesExecute...), append(bootNodesPropose, bootNodesValidate...)...)

	bootNodeMap := make(map[string]struct{})
	for _, bootNode := range bootNodesAll {
		if _, ok := bootNodeMap[bootNode]; ok {
			continue
		}

		peerInfo, err := p2p.getAddrInfo(bootNode)
		if err != nil {
			return false
		}

		if peerInfo.ID == p {
			return true
		}

		bootNodeMap[bootNode] = struct{}{}
	}

	return false
}

func (p2p *P2PService) defaultPubSubOptions() []pubsub.Option {
	var ipcoloWhitelist []*net.IPNet
	for _, cidr := range p2p.config.PubSub.IPColocationWhitelist {
		_, ipnet, err := net.ParseCIDR(cidr)
		if err != nil {
			panic(fmt.Sprintf("error parsing IPColocation subnet %s: %v", cidr, err))
		}
		ipcoloWhitelist = append(ipcoloWhitelist, ipnet)
	}

	topicParams := map[string]*pubsub.TopicScoreParams{
		tpnetprotoc.PubSubProtocolID_BlockInfo: {
			// expected 10 blocks/min
			TopicWeight: 0.1, // max cap is 50, max mesh penalty is -10, single invalid message is -100

			// 1 tick per second, maxes at 1 after 1 hour
			TimeInMeshWeight:  0.00027, // ~1/3600
			TimeInMeshQuantum: time.Second,
			TimeInMeshCap:     1,

			// deliveries decay after 1 hour, cap at 100 blocks
			FirstMessageDeliveriesWeight: 5, // max value is 500
			FirstMessageDeliveriesDecay:  pubsub.ScoreParameterDecay(time.Hour),
			FirstMessageDeliveriesCap:    100, // 100 blocks in an hour

			InvalidMessageDeliveriesWeight: -1000,
			InvalidMessageDeliveriesDecay:  pubsub.ScoreParameterDecay(time.Hour),
		},
		tpnetprotoc.PubSubProtocolID_Msgs: {
			// expected > 1 tx/second
			TopicWeight: 0.1, // max cap is 5, single invalid message is -100

			// 1 tick per second, maxes at 1 hour
			TimeInMeshWeight:  0.0002778, // ~1/3600
			TimeInMeshQuantum: time.Second,
			TimeInMeshCap:     1,

			FirstMessageDeliveriesWeight: 0.5,
			FirstMessageDeliveriesDecay:  pubsub.ScoreParameterDecay(10 * time.Minute),
			FirstMessageDeliveriesCap:    100, // 100 messages in 10 minutes

			InvalidMessageDeliveriesWeight: -1000,
			InvalidMessageDeliveriesDecay:  pubsub.ScoreParameterDecay(time.Hour),
		},
	}

	options := []pubsub.Option{
		pubsub.WithMaxMessageSize(tpnetprotoc.PubSubMaxMsgSize),
		pubsub.WithFloodPublish(true),
		pubsub.WithPeerScore(
			&pubsub.PeerScoreParams{
				AppSpecificScore: func(p peer.ID) float64 {
					if p2p.isBootNode(p) && !p2p.config.PubSub.ISSeedPeer {
						return 2500
					}
					return 0
				},
				AppSpecificWeight: 1,

				// This sets the IP colocation threshold to 5 peers before we apply penalties
				IPColocationFactorThreshold: 5,
				IPColocationFactorWeight:    -100,
				IPColocationFactorWhitelist: ipcoloWhitelist,

				// P7: behavioural penalties, decay after 1hr
				BehaviourPenaltyThreshold: 6,
				BehaviourPenaltyWeight:    -10,
				BehaviourPenaltyDecay:     pubsub.ScoreParameterDecay(time.Hour),

				DecayInterval: pubsub.DefaultDecayInterval,
				DecayToZero:   pubsub.DefaultDecayToZero,

				// this retains non-positive scores for 6 hours
				RetainScore: 6 * time.Hour,

				// topic parameters
				Topics: topicParams,
			},
			&pubsub.PeerScoreThresholds{
				GossipThreshold:             GossipScoreThreshold,
				PublishThreshold:            PublishScoreThreshold,
				GraylistThreshold:           GraylistScoreThreshold,
				AcceptPXThreshold:           AcceptPXScoreThreshold,
				OpportunisticGraftThreshold: OpportunisticGraftScoreThreshold,
			},
		),
		pubsub.WithPeerScoreInspect(p2p.peerScoreCache.Update, 10*time.Second),
	}

	// enable Peer eXchange on bootstrappers
	if p2p.config.PubSub.ISSeedPeer {
		// turn off the mesh in bootstrappers -- only do gossip and PX
		pubsub.GossipSubD = 0
		pubsub.GossipSubDscore = 0
		pubsub.GossipSubDlo = 0
		pubsub.GossipSubDhi = 0
		pubsub.GossipSubDout = 0
		pubsub.GossipSubDlazy = 64
		pubsub.GossipSubGossipFactor = 0.25
		pubsub.GossipSubPruneBackoff = 5 * time.Minute
		// turn on PX
		options = append(options, pubsub.WithPeerExchange(true))
	}

	// direct peers
	if p2p.config.PubSub.DirectPeers != nil {
		var directPeerInfo []peer.AddrInfo

		for _, addr := range p2p.config.PubSub.DirectPeers {
			a, err := ma.NewMultiaddr(addr)
			if err != nil {
				directPeerInfo = nil
				panic(fmt.Sprintf("Invalid DirectPeer: addr %s, err %v", addr, err))
			}

			pi, err := peer.AddrInfoFromP2pAddr(a)
			if err != nil {
				panic(fmt.Sprintf("AddrInfoFromP2pAddr err: addr %s, err %v", addr, err))
			}

			directPeerInfo = append(directPeerInfo, *pi)
		}

		options = append(options, pubsub.WithDirectPeers(directPeerInfo))
	}

	return options
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

func (p2p *P2PService) getBootNodes(ctx context.Context) ([]string, []string, []string, []string) {
	var bootNodes []string
	var bootNodesExecute []string
	var bootNodesPropose []string
	var bootNodesValidate []string

	if ctx.Value(tpnetcmn.NetContextKey_BOOTNODES) != nil {
		bootNodes = ctx.Value(tpnetcmn.NetContextKey_BOOTNODES).([]string)
	}
	if ctx.Value(tpnetcmn.NetContextKey_BOOTNODES_EXECUTE) != nil {
		bootNodesExecute = ctx.Value(tpnetcmn.NetContextKey_BOOTNODES_EXECUTE).([]string)
	}
	if ctx.Value(tpnetcmn.NetContextKey_BOOTNODES_PROPOSE) != nil {
		bootNodesPropose = ctx.Value(tpnetcmn.NetContextKey_BOOTNODES_PROPOSE).([]string)
	}
	if ctx.Value(tpnetcmn.NetContextKey_BOOTNODES_VALIDATE) != nil {
		bootNodesValidate = ctx.Value(tpnetcmn.NetContextKey_BOOTNODES_PROPOSE).([]string)
	}

	return bootNodes, bootNodesExecute, bootNodesPropose, bootNodesValidate
}

func (p2p *P2PService) createDHTService(ctx context.Context, p2pLog tplog.Logger, h host.Host) error {
	var dhtOptions []dht.Option
	var dhtOptionsExecute []dht.Option
	var dhtOptionsPropose []dht.Option
	var dhtOptionsValidate []dht.Option

	bootNodes, bootNodesExecute, bootNodesPropose, bootNodesValidate := p2p.getBootNodes(ctx)

	if len(bootNodes) > 0 {
		dhtOptions = append(dhtOptions, p2p.withBootPeers(bootNodes))
	}
	dhtOptions = append(dhtOptions, p2p.defaultDHTOptions()...)

	if len(bootNodesExecute) > 0 {
		dhtOptionsExecute = append(dhtOptionsExecute, p2p.withBootPeers(bootNodesExecute))
	}
	dhtOptionsExecute = append(dhtOptionsExecute, p2p.defaultDHTOptionsExecute()...)

	if len(bootNodesPropose) > 0 {
		dhtOptionsPropose = append(dhtOptionsPropose, p2p.withBootPeers(bootNodesPropose))
	}
	dhtOptionsPropose = append(dhtOptionsPropose, p2p.defaultDHTOptionsPropose()...)

	if len(bootNodesValidate) > 0 {
		dhtOptionsValidate = append(dhtOptionsValidate, p2p.withBootPeers(bootNodesValidate))
	}
	dhtOptionsValidate = append(dhtOptionsValidate, p2p.defaultDHTOptionsValidate()...)

	if p2p.netActiveNode != nil {
		dhtOptionsExecute = append(dhtOptionsExecute, dht.RoutingTableFilter(func(dht interface{}, p peer.ID) bool {
			activeExcutors, err := p2p.netActiveNode.GetActiveExecutorIDs()
			if err != nil {
				p2pLog.Errorf("Can't get active excutors: %v", err)
			}

			p2pLog.Debugf("Current active excutors: %v", activeExcutors)

			if len(activeExcutors) > 0 {
				return tpcmm.IsContainString(p.String(), activeExcutors)
			}

			return false
		}))

		dhtOptionsPropose = append(dhtOptionsPropose, dht.RoutingTableFilter(func(dht interface{}, p peer.ID) bool {
			activeProposers, err := p2p.netActiveNode.GetActiveProposerIDs()
			if err != nil {
				p2pLog.Errorf("Can't get active proposers: %v", err)
			}

			p2pLog.Debugf("Current active proposers: %v", activeProposers)

			if len(activeProposers) > 0 {
				return tpcmm.IsContainString(p.String(), activeProposers)
			}

			return false
		}))

		dhtOptionsValidate = append(dhtOptionsValidate, dht.RoutingTableFilter(func(dht interface{}, p peer.ID) bool {
			activeValidates, err := p2p.netActiveNode.GetActiveValidatorIDs()
			if err != nil {
				p2pLog.Errorf("Can't get active validates: %v", err)
			}

			p2pLog.Debugf("Current active validates: %v", activeValidates)

			if len(activeValidates) > 0 {
				return tpcmm.IsContainString(p.String(), activeValidates)
			}

			return false
		}))

	}

	dhtOptsMap := map[DHTServiceType][]dht.Option{
		DHTServiceType_General:  dhtOptions,
		DHTServiceType_Execute:  dhtOptionsExecute,
		DHTServiceType_Propose:  dhtOptionsPropose,
		DHTServiceType_Validate: dhtOptionsValidate,
	}

	p2p.dhtServices = make(map[DHTServiceType]*P2PDHTService)
	for dhtSType, dhtOpts := range dhtOptsMap {
		dht, err := dht.New(ctx, h, dhtOpts...)
		if err != nil {
			p2pLog.Errorf("create p2p dht err: %v", err)
			return err
		}

		err = dht.Bootstrap(ctx)
		if err != nil {
			p2pLog.Errorf("dht bootstrap err: %v", err)
			return err
		}

		p2p.dhtServices[dhtSType] = NewP2PDHTService(ctx, p2pLog, dht)
	}

	return nil
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

func (p2p *P2PService) DHTServiceOfProtocol(protocolID string) (*P2PDHTService, error) {
	if strings.Contains(protocolID, tpnetprotoc.P2PProtocolPrefix) {
		return p2p.dhtServices[DHTServiceType_General], nil
	}

	if strings.Contains(protocolID, tpnetprotoc.P2PProtocolExecutePrefix) {
		return p2p.dhtServices[DHTServiceType_Execute], nil
	}

	if strings.Contains(protocolID, tpnetprotoc.P2PProtocolProposePrefix) {
		return p2p.dhtServices[DHTServiceType_Propose], nil
	}

	if strings.Contains(protocolID, tpnetprotoc.P2PProtocolValidatePrefix) {
		return p2p.dhtServices[DHTServiceType_Validate], nil
	}

	return nil, fmt.Errorf("unknown protocolID %s", protocolID)
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

		dhtService, err := p2p.DHTServiceOfProtocol(protocolID)
		if err != nil {
			return err
		}

		switch rStrategy {
		case tpnetcmn.RouteStrategy_Default:
			peerIDList, _ = dhtService.GetAllPeerIDs()
		case tpnetcmn.RouteStrategy_NearestBucket:
			peerIDList, _ = dhtService.GetNearestPeerIDs(p2p.host.ID())
		case tpnetcmn.RouteStrategy_BucketsWithFactor:
			peerIDList, _ = dhtService.GetPeersWithFactor()
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

func (p2p *P2PService) SendWithResponse(ctx context.Context, protocolID string, moduleName string, data []byte) ([]message.SendResponse, error) {
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

		dhtService, err := p2p.DHTServiceOfProtocol(protocolID)
		if err != nil {
			return nil, err
		}

		switch rStrategy {
		case tpnetcmn.RouteStrategy_Default:
			peerIDList, _ = dhtService.GetAllPeerIDs()
		case tpnetcmn.RouteStrategy_NearestBucket:
			peerIDList, _ = dhtService.GetNearestPeerIDs(p2p.host.ID())
		case tpnetcmn.RouteStrategy_BucketsWithFactor:
			peerIDList, _ = dhtService.GetPeersWithFactor()
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

	respCh := make(chan *message.SendResponse, len(peerIDList))
	var respList []message.SendResponse

	forceEndChs := make(map[string]chan struct{}, len(peerIDList))
	for _, id := range peerIDList {
		forceEndChs[id.String()] = make(chan struct{})
	}

	for _, id := range peerIDList {
		go func(peerID peer.ID) {
			defer func() {
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

			sendResp := &message.SendResponse{NodeID: peerID.String()}

			stream, err := p2p.host.NewStream(
				network.WithNoDial(ctx, "should already have connection"),
				peerID,
				protocol.ID(protocolID))
			if err != nil {
				p2p.log.Errorf("failed to open stream to peer: %v", err)
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
			if err = p2p.streamService.writeMessage(stream, msg); err != nil {
				_ = stream.SetWriteDeadline(time.Time{})
				err = fmt.Errorf("Stream sendMessage error: stream=%s, err=%s", stream.ID(), err.Error())
				p2p.log.Errorf("%v", err)
				sendResp.Err = err
				respCh <- sendResp
				return
			}
			_ = stream.SetWriteDeadline(time.Time{}) // clear deadline // FIXME: Needs
			//  its own API (https://github.com/libp2p/go-libp2p-core/issues/162).

			resp, err := p2p.streamService.readMessage(stream)
			if err == nil {
				sendResp.RespData = resp.Data
				respCh <- sendResp
				p2p.host.ConnManager().TagPeer(peerID, "SendSync", 25)
			} else {
				err = fmt.Errorf("read resp error %v", err)
				p2p.log.Errorf("%v", err)
				sendResp.Err = err
				respCh <- sendResp
			}

			return
		}(id)
	}

	threshold := 1
	if ctx.Value(tpnetcmn.NetContextKey_RespThreshold) != nil {
		thresholdR := ctx.Value(tpnetcmn.NetContextKey_RespThreshold).(float32)
		threshold = int(float32(len(peerIDList)) * thresholdR)
	}

	respLen := 0
	for r := range respCh {
		respLen++
		respList = append(respList, *r)
		if respLen >= cap(respCh) || respLen >= threshold {
			break
		}
	}

	dur := time.Since(startTime)
	p2p.log.Debugf("received resp %d after %d ms", len(respCh), dur.Microseconds())

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
	var maP2PAddrs []ma.Multiaddr
	for _, addr := range listenAddr {
		maAddr, err := ma.NewMultiaddr(addr)
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

func (p2p *P2PService) Subscribe(ctx context.Context, topic string, localIgnore bool, validators ...message.PubSubMessageValidator) error {
	return p2p.pubsubService.Subscribe(ctx, topic, localIgnore, validators...)
}

func (p2p *P2PService) UnSubscribe(topic string) error {
	return p2p.pubsubService.UnSubscribe(topic)
}

func (p2p *P2PService) Publish(ctx context.Context, toModuleNames []string, topic string, data []byte) error {
	return p2p.pubsubService.Publish(ctx, toModuleNames, topic, data)
}

func (p2p *P2PService) Start() {
	timer := time.NewTicker(5 * time.Second)
	defer timer.Stop()

	for {
		select {
		case <-timer.C:
			peerCount := len(p2p.ConnectedPeers())
			if peerCount == 0 {
				if len(p2p.config.Connection.SeedPeers) == 0 {
					p2p.log.Warn("Connected 0 peer while no any seed peer")
					break
				}

				wg := sync.WaitGroup{}
				for _, seedPeer := range p2p.config.Connection.SeedPeers {
					wg.Add(1)
					go func(sPeer string) {
						defer wg.Done()

						peerAddr, err := peer.AddrInfoFromString(sPeer)
						if err != nil {
							p2p.log.Errorf("Invalid seed peer addr %s: %v", sPeer, err)
							return
						}

						if err = p2p.host.Connect(p2p.ctx, *peerAddr); err != nil {
							p2p.log.Errorf("Failed to connect seed peer addr %s: %v", sPeer, err)
							return
						}
					}(seedPeer.NetAddrString)
				}
				wg.Wait()
			}

			if peerCount < p2p.config.Connection.LowWater {
				if err := p2p.dhtServices[DHTServiceType_General].dht.Bootstrap(p2p.ctx); err != nil {
					p2p.log.Errorf("General dht bootstrap err: %v", err)
					continue
				}
			}
		case <-p2p.ctx.Done():

		}
	}
}

func (p2p *P2PService) Close() {
	for _, dhtService := range p2p.dhtServices {
		dhtService.Close()
	}

	p2p.host.Close()
}

func (p2p *P2PService) ConnectedPeers() []*tpnetcmn.RemotePeer {
	conns := p2p.host.Network().Conns()

	var remotePeers []*tpnetcmn.RemotePeer
	for _, conn := range conns {
		remotePeer := &tpnetcmn.RemotePeer{
			ID:   conn.RemotePeer().String(),
			Addr: conn.RemoteMultiaddr().String(),
		}

		remotePeers = append(remotePeers, remotePeer)
	}

	return remotePeers
}

func (p2p *P2PService) Connectedness(nodeID string) (tpnetcmn.Connectedness, error) {
	peerID, err := peer.IDFromString(nodeID)
	if err != nil {
		return tpnetcmn.NotConnected, err
	}

	return tpnetcmn.Connectedness(p2p.host.Network().Connectedness(peerID)), nil
}

func (p2p *P2PService) PubSubScores() []tpnetcmn.PubsubScore {
	var psScores []tpnetcmn.PubsubScore

	for peerID, psSnapShot := range p2p.peerScoreCache.Fetch() {
		psScore := tpnetcmn.PubsubScore{
			ID:    peerID.String(),
			Score: psSnapShot,
		}

		psScores = append(psScores, psScore)
	}

	return psScores
}

func (p2p *P2PService) NatState() (*tpnetcmn.NatInfo, error) {
	autonat := p2p.host.(*basichost.BasicHost).GetAutoNat()

	if autonat == nil {
		return &tpnetcmn.NatInfo{
			Reachability: tpnetcmn.ReachabilityUnknown,
		}, nil
	}

	var maddr string
	if autonat.Status() == network.ReachabilityPublic {
		pa, err := autonat.PublicAddr()
		if err != nil {
			return nil, err
		}
		maddr = pa.String()
	}

	return &tpnetcmn.NatInfo{
		Reachability: tpnetcmn.Reachability(autonat.Status()),
		PublicAddr:   maddr,
	}, nil
}

func (p2p *P2PService) PeerDetailInfo(nodeID string) (*tpnetcmn.PeerDetail, error) {
	var peerDetail tpnetcmn.PeerDetail
	peerID, err := peer.IDFromString(nodeID)
	if err != nil {
		return nil, err
	}

	agent, err := p2p.host.Peerstore().Get(peerID, "AgentVersion")
	if err == nil {
		peerDetail.AgentVersion = agent.(string)
	} else {
		return nil, err
	}

	for _, a := range p2p.host.Peerstore().Addrs(peerID) {
		peerDetail.ConnectedAddrs = append(peerDetail.ConnectedAddrs, a.String())
	}
	sort.Strings(peerDetail.ConnectedAddrs)

	protocols, err := p2p.host.Peerstore().GetProtocols(peerID)
	if err == nil {
		sort.Strings(protocols)
		peerDetail.SupportedProtocols = protocols
	} else {
		return nil, err
	}

	if cm := p2p.host.ConnManager().GetTagInfo(peerID); cm != nil {
		peerDetail.ConnInfos = &tpnetcmn.ConnInfos{
			FirstSeen: cm.FirstSeen,
			Value:     cm.Value,
			Tags:      cm.Tags,
			Conns:     cm.Conns,
		}
	}

	return &peerDetail, nil
}

func (p2p *P2PService) FindPeer(ctx context.Context, nodeID string) (string, error) {
	peerID, err := peer.IDFromString(nodeID)
	if err != nil {
		return "", err
	}

	executorIDs, _ := p2p.netActiveNode.GetActiveExecutorIDs()
	if tpcmm.IsContainString(nodeID, executorIDs) {
		maAddr, err := p2p.dhtServices[DHTServiceType_Execute].dht.FindPeer(ctx, peerID)
		if err != nil {
			return "", err
		}
		return maAddr.String(), nil
	}

	proposerIDs, _ := p2p.netActiveNode.GetActiveProposerIDs()
	if tpcmm.IsContainString(nodeID, proposerIDs) {
		maAddr, err := p2p.dhtServices[DHTServiceType_Propose].dht.FindPeer(ctx, peerID)
		if err != nil {
			return "", err
		}
		return maAddr.String(), nil
	}

	validatorIDs, _ := p2p.netActiveNode.GetActiveValidatorIDs()
	if tpcmm.IsContainString(nodeID, validatorIDs) {
		maAddr, err := p2p.dhtServices[DHTServiceType_Validate].dht.FindPeer(ctx, peerID)
		if err != nil {
			return "", err
		}
		return maAddr.String(), nil
	}

	maAddr, err := p2p.dhtServices[DHTServiceType_General].dht.FindPeer(ctx, peerID)
	if err != nil {
		return "", err
	}
	return maAddr.String(), nil
}

func (p2p *P2PService) ConnectToNode(ctx context.Context, nodeNetAddr string) error {
	peerAddr, err := peer.AddrInfoFromString(nodeNetAddr)
	if err != nil {
		return err
	}

	if swrm, ok := p2p.host.Network().(*swarm.Swarm); ok {
		swrm.Backoff().Clear(peerAddr.ID)
	}

	return p2p.host.Connect(ctx, *peerAddr)
}

func (p2p *P2PService) DisConnectWithNode(nodeID string) error {
	peerID, err := peer.IDFromString(nodeID)
	if err != nil {
		return err
	}

	return p2p.host.Network().ClosePeer(peerID)
}
