package system_interaction

import (
	"context"
	"github.com/AsynkronIT/protoactor-go/actor"
	"github.com/TopiaNetwork/topia/configuration"
	"github.com/stretchr/testify/require"
	"testing"
	"time"

	"github.com/TopiaNetwork/topia/codec"
	"github.com/TopiaNetwork/topia/integration/mock"
	tplog "github.com/TopiaNetwork/topia/log"
	tplogcmm "github.com/TopiaNetwork/topia/log/common"
	"github.com/TopiaNetwork/topia/network"
	tpnetcmn "github.com/TopiaNetwork/topia/network/common"
	"github.com/TopiaNetwork/topia/network/protocol"
	"github.com/TopiaNetwork/topia/sync"
)

func TestBlockRequest(t *testing.T) {
	ctx := context.Background()
	testLog, _ := tplog.CreateMainLogger(tplogcmm.InfoLevel, tplog.JSONFormat, tplog.StdErrOutput, "")

	netConfig := configuration.GetConfiguration().NetConfig

	sysActor1 := actor.NewActorSystem()
	network1 := network.NewNetwork(ctx, testLog, netConfig, sysActor1, "/ip4/127.0.0.1/tcp/41000", "topia1", nil)
	testLog.Infof("network1 id=%s, addrs=%v", network1.ID(), network1.ListenAddr())

	sysActor2 := actor.NewActorSystem()
	network2 := network.NewNetwork(ctx, testLog, netConfig, sysActor2, "/ip4/127.0.0.1/tcp/41001", "topia2", nil)
	testLog.Infof("network1 id=%s, addrs=%v", network2.ID(), network2.ListenAddr())

	err := network1.Connect(network2.ListenAddr())
	require.Equal(t, nil, err)

	syncer1 := sync.NewSyncer(tplogcmm.InfoLevel, testLog, codec.CodecType_PROTO)
	syncer1.Start(sysActor1, network1)
	syncer1.UpdateHandler(&mock.MockSyncHandler{
		Log: testLog,
	})

	syncer2 := sync.NewSyncer(tplogcmm.InfoLevel, testLog, codec.CodecType_PROTO)
	syncer2.Start(sysActor2, network2)
	syncer2.UpdateHandler(&mock.MockSyncHandler{
		Log: testLog,
	})

	time.Sleep(10 * time.Second)

	blockReq := sync.BlockRequest{}

	brData, _ := syncer1.Marshaler().Marshal(&blockReq)

	syncMsg := sync.SyncMessage{
		MsgType: sync.SyncMessage_BlockRequest,
		Data:    brData,
	}

	syncData, _ := syncer1.Marshaler().Marshal(&syncMsg)

	ctx1 := context.WithValue(ctx, tpnetcmn.NetContextKey_RouteStrategy, tpnetcmn.RouteStrategy_NearestBucket)

	resps, err := network1.SendWithResponse(ctx1, protocol.SyncProtocolID_Block, "sync", syncData)
	require.Equal(t, nil, err)
	require.Equal(t, 1, len(resps))

	var resp sync.SyncMessage
	err = syncer1.Marshaler().Unmarshal(resps[0].RespData, &resp)
	require.Equal(t, nil, err)
	require.Equal(t, sync.SyncMessage_BlockResponse, resp.MsgType)

	var blockResp sync.BlockResponse
	err = syncer1.Marshaler().Unmarshal(resp.Data, &blockResp)
	require.Equal(t, nil, err)
	require.Equal(t, 100, int(blockResp.Height))
}
