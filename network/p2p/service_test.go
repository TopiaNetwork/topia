package p2p

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	tplog "github.com/TopiaNetwork/topia/log"
	tplogcmm "github.com/TopiaNetwork/topia/log/common"
	tpnetcmn "github.com/TopiaNetwork/topia/network/common"
	"github.com/TopiaNetwork/topia/network/protocol"
)

const ticksForAssertEventually = 100 * time.Millisecond

func TestSend(t *testing.T) {
	testLog, _ := tplog.CreateMainLogger(tplogcmm.InfoLevel, tplog.JSONFormat, tplog.StdErrOutput, "")

	p2p1 := NewP2PService(context.Background(), testLog, nil, "/ip4/127.0.0.1/tcp/41000", "topia1", NewNetworkActiveNodeMock())

	testLog.Infof("p2p1 id=%s", p2p1.ID().String())

	p2p2 := NewP2PService(context.Background(), testLog, nil, "/ip4/127.0.0.1/tcp/41001", "topia2", NewNetworkActiveNodeMock())

	testLog.Infof("p2p2 id=%s", p2p2.ID().String())

	ctx := context.Background()
	err := p2p1.Connect(p2p2.ListenAddr())
	assert.Equal(t, nil, err)
	err = p2p1.Send(context.WithValue(ctx, tpnetcmn.NetContextKey_PeerList, []string{p2p2.ID().String()}), protocol.AsyncSendProtocolID, "", nil)
	assert.Equal(t, nil, err)

	time.Sleep(30 * time.Second)
}

func sendByDHT(t *testing.T, routeStrategy tpnetcmn.RouteStrategy) {
	testLog, _ := tplog.CreateMainLogger(tplogcmm.InfoLevel, tplog.JSONFormat, tplog.StdErrOutput, "")

	p2p1 := NewP2PService(context.Background(), testLog, nil, "/ip4/127.0.0.1/tcp/41000", "topia1", NewNetworkActiveNodeMock())

	testLog.Infof("p2p1 id=%s", p2p1.ID().String())

	p2p2 := NewP2PService(context.Background(), testLog, nil, "/ip4/127.0.0.1/tcp/41001", "topia2", NewNetworkActiveNodeMock())

	testLog.Infof("p2p2 id=%s", p2p2.ID().String())

	p2p1.Connect(p2p2.ListenAddr())

	require.Eventually(t, func() bool {
		return p2p1.dhtServices[DHTServiceType_General].dht.RoutingTable().Find(p2p2.ID()) != ""
	}, time.Second*5, ticksForAssertEventually, "dht servers p2p1 failed to connect")

	require.Eventually(t, func() bool {
		return p2p2.dhtServices[DHTServiceType_General].dht.RoutingTable().Find(p2p1.ID()) != ""
	}, time.Second*5, ticksForAssertEventually, "dht servers p2p2 failed to connect")

	ctx := context.Background()
	ctx1 := context.WithValue(ctx, tpnetcmn.NetContextKey_RouteStrategy, routeStrategy)

	err := p2p1.Send(ctx1, protocol.AsyncSendProtocolID, "", nil)
	assert.Equal(t, nil, err)

	err = p2p1.Send(ctx1, protocol.AsyncSendProtocolID, "", nil)
	assert.Equal(t, nil, err)

	time.Sleep(30 * time.Second)
}

func TestSendByDHTDefaultStrategy(t *testing.T) {
	sendByDHT(t, tpnetcmn.RouteStrategy_Default)
}

func TestSendByDHTNearestStrategy(t *testing.T) {
	sendByDHT(t, tpnetcmn.RouteStrategy_NearestBucket)
}

func TestSendByDHTBucketsWithFactorStrategy(t *testing.T) {
	sendByDHT(t, tpnetcmn.RouteStrategy_BucketsWithFactor)
}

func TestSendWithMultiProtocols(t *testing.T) {
	testLog, _ := tplog.CreateMainLogger(tplogcmm.InfoLevel, tplog.JSONFormat, tplog.StdErrOutput, "")

	p2p1 := NewP2PService(context.Background(), testLog, nil, "/ip4/127.0.0.1/tcp/41000", "topia1", NewNetworkActiveNodeMock())

	testLog.Infof("p2p1 id=%s", p2p1.ID().String())

	p2p2 := NewP2PService(context.Background(), testLog, nil, "/ip4/127.0.0.1/tcp/41001", "topia2", NewNetworkActiveNodeMock())

	testLog.Infof("p2p2 id=%s", p2p2.ID().String())

	p2p3AvtiveNodes := NewNetworkActiveNodeMock()
	p2p3AvtiveNodes.addActiveValidator(p2p2.ID().String())
	p2p3AvtiveNodes.addActiveProposer(p2p1.ID().String())
	p2p3 := NewP2PService(context.Background(), testLog, nil, "/ip4/127.0.0.1/tcp/41002", "topia3", p2p3AvtiveNodes)

	testLog.Infof("p2p3 id=%s", p2p2.ID().String())

	p2p1.Connect(p2p2.ListenAddr())
	p2p1.Connect(p2p3.ListenAddr())
	p2p2.Connect(p2p3.ListenAddr())

	require.Eventually(t, func() bool {
		return p2p1.dhtServices[DHTServiceType_General].dht.RoutingTable().Find(p2p2.ID()) != ""
	}, time.Second*5, ticksForAssertEventually, "dht servers p2p1 failed to connect")

	require.Eventually(t, func() bool {
		return p2p2.dhtServices[DHTServiceType_General].dht.RoutingTable().Find(p2p1.ID()) != ""
	}, time.Second*5, ticksForAssertEventually, "dht servers p2p2 failed to connect")

	assert.Equal(t, 2, p2p1.dhtServices[DHTServiceType_General].dht.RoutingTable().Size())

	assert.Equal(t, 2, p2p2.dhtServices[DHTServiceType_General].dht.RoutingTable().Size())

	assert.Equal(t, 2, p2p3.dhtServices[DHTServiceType_General].dht.RoutingTable().Size())
	assert.Equal(t, 0, p2p3.dhtServices[DHTServiceType_Execute].dht.RoutingTable().Size())
	assert.Equal(t, 1, p2p3.dhtServices[DHTServiceType_Propose].dht.RoutingTable().Size())
	assert.Equal(t, 1, p2p3.dhtServices[DHTServiceType_Validate].dht.RoutingTable().Size())

	ctx := context.Background()
	ctx1 := context.WithValue(ctx, tpnetcmn.NetContextKey_RouteStrategy, tpnetcmn.RouteStrategy_NearestBucket)

	err := p2p1.Send(ctx1, protocol.AsyncSendProtocolID, "", nil)
	assert.Equal(t, nil, err)

	err = p2p1.Send(ctx1, protocol.SyncProtocolID_Block, "", nil)
	assert.Equal(t, nil, err)

	err = p2p1.Send(ctx1, protocol.SyncProtocolID_Msg, "", nil)
	assert.Equal(t, nil, err)

	err = p2p1.Send(ctx1, protocol.HeatBeatPtotocolID, "", nil)
	assert.Equal(t, nil, err)

	time.Sleep(30 * time.Second)
}
