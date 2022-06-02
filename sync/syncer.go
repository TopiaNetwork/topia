package sync

import (
	"context"
	"github.com/AsynkronIT/protoactor-go/actor"
	"github.com/TopiaNetwork/topia/codec"
	"github.com/TopiaNetwork/topia/common"
	"github.com/TopiaNetwork/topia/ledger"
	tplog "github.com/TopiaNetwork/topia/log"
	tplogcmm "github.com/TopiaNetwork/topia/log/common"
	"github.com/TopiaNetwork/topia/network"
	"github.com/TopiaNetwork/topia/network/protocol"
	"github.com/TopiaNetwork/topia/service"
	"sync"
	"time"
)

const (
	MOD_NAME                 = "sync"
	MOD_ACTOR_NAME           = "sync_actor"
	SyncHeartBeatInterval    = 1000 * time.Millisecond // Time interval to force syncs, even if few peers are available
	RequestBlockInterval     = 100 * time.Millisecond
	RequestEpochInterval     = 100 * time.Millisecond
	RequestNodeStateInterval = 100 * time.Millisecond
)

type Syncer interface {
	UpdateHandler(handler SyncHandler)

	Marshaler() codec.Marshaler

	Start(sysActor *actor.ActorSystem, network network.Network) error

	Stop()
}

type syncer struct {
	log       tplog.Logger
	level     tplogcmm.LogLevel
	handler   *syncHandler
	marshaler codec.Marshaler
	ctx       context.Context

	savedBlockHeights *common.IdBitmap

	curBlockHeight     uint64
	blockSyncQueue     *common.LockFreePriorityQueue
	query              SyncServant
	wg                 sync.WaitGroup
	SyncHeartBeatTimer *time.Timer //Forced synchronization timer
	chanSyncIsDoing    chan error  //chan is nil if sync is done
	chanQuitBlockSync  chan error
}

func NewSyncer(level tplogcmm.LogLevel, log tplog.Logger, marshaler codec.Marshaler, ctx context.Context, ledger ledger.Ledger,
	stateQueryService service.StateQueryService,
	blockService service.BlockService) Syncer {
	syncLog := tplog.CreateModuleLogger(level, MOD_NAME, log)
	sy := &syncer{
		log:       syncLog,
		level:     level,
		marshaler: marshaler,
		query:     NewSyncServant(stateQueryService, blockService),
		ctx:       ctx,
	}
	sy.handler = NewSyncHandler(sy.log, ctx, sy.marshaler, ledger, stateQueryService, blockService)
	return sy
}

func (sa *syncer) UpdateHandler(handler SyncHandler) {

}

func (sa *syncer) Marshaler() codec.Marshaler {
	return sa.marshaler
}

func (sa *syncer) handleBlockRequest(msg *BlockRequest) error {
	return sa.handler.HandleBlockRequest(msg)
}

func (sa *syncer) handleBlockResponse(msg *BlockResponse) error {
	return sa.handler.HandleBlockResponse(msg)
}
func (sa *syncer) handleStateRequest(msg *StateRequest) error {
	return sa.handler.HandleStateRequest(msg)
}

func (sa *syncer) handleStateResponse(msg *StateResponse) error {
	return sa.handler.HandleStateResponse(msg)
}

//
//func (sa *syncer) handleEpochRequest(msg *EpochRequest) error {
//	return sa.handler.HandleEpochRequest(msg)
//}
//
func (sa *syncer) handleEpochResponse(msg *EpochResponse) error {
	return sa.handler.HandleEpochResponse(msg)
}

//
//func (sa *syncer) handleNodeRequest(msg *NodeRequest) error {
//	return sa.handler.HandleNodeRequest(msg)
//}
//
//func (sa *syncer) handleNodeResponse(msg *NodeResponse) error {
//	return sa.handler.HandleNodeResponse(msg)
//}

func (sa *syncer) Start(sysActor *actor.ActorSystem, network network.Network) error {
	actorPID, err := CreateSyncActor(sa.level, sa.log, sysActor, sa)
	if err != nil {
		sa.log.Panicf("CreateSyncActor error: %v", err)
		return err
	}
	network.RegisterModule(MOD_NAME, actorPID, sa.marshaler)
	sa.loop()

	go sa.handler.blockDownloader.loopFetchBlocks()
	go sa.handler.blockDownloader.loopDoBlockSync()

	go sa.handler.epochDownloader.loopFetchEpochs()
	go sa.handler.epochDownloader.loopDoEpochSync()
	return nil
}

func (sa *syncer) loop() {
	sa.SyncHeartBeatTimer = time.NewTimer(SyncHeartBeatInterval)
	defer sa.SyncHeartBeatTimer.Stop()
	for {
		bytesSyncHeartBeat := sa.getCurrentHeartBeatByte()

		select {
		case <-sa.chanSyncIsDoing:
			sa.chanSyncIsDoing = nil
			sa.SyncHeartBeatTimer.Reset(SyncHeartBeatInterval)

		case <-sa.SyncHeartBeatTimer.C:
			sa.query.Send(sa.ctx, protocol.HeartBeatPtotocolID, MOD_NAME, bytesSyncHeartBeat)
			sa.SyncHeartBeatTimer.Reset(SyncHeartBeatInterval)
		case <-sa.handler.quitSync:
			//quit synchronization requires stopping the insertion of data into the blockchain
			//and then stopping the downloader.
			if sa.chanSyncIsDoing != nil {
				<-sa.chanSyncIsDoing
			}
			return
		}
	}

}

func (sa *syncer) getCurrentHeartBeatByte() []byte {
	curHeight, err := sa.query.GetLatestHeight()
	if err != nil {
		sa.log.Errorf("query current Height Error:", err)
	}
	curStateRoot, err := sa.query.StateRoot()
	if err != nil {
		sa.log.Errorf("query current account state root Error:", err)
	}
	curStateVersion, err := sa.query.StateLatestVersion()
	if err != nil {
		sa.log.Errorf("query current chain state root Error:", err)
	}

	heartBeatRequest := &HeartBeatRequest{
		BlockHeight:          curHeight,
		StateRoot:            curStateRoot,
		SateVersion:          curStateVersion,
		XXX_NoUnkeyedLiteral: struct{}{},
		XXX_unrecognized:     nil,
		XXX_sizecache:        0,
	}

	data, _ := sa.marshaler.Marshal(heartBeatRequest)
	syncMsg := SyncMessage{
		MsgType: SyncMessage_HeartBeatRequest,
		Data:    data,
	}
	syncData, _ := sa.marshaler.Marshal(&syncMsg)
	return syncData
}

func (sa *syncer) dispatch(context context.Context, data []byte) {
	var syncMsg SyncMessage
	err := sa.marshaler.Unmarshal(data, &syncMsg)
	if err != nil {
		sa.log.Errorf("Syncer receive invalid data %v", data)
		return
	}

	switch syncMsg.MsgType {
	case SyncMessage_BlockRequest:
		var msg BlockRequest
		err := sa.marshaler.Unmarshal(syncMsg.Data, &msg)
		if err != nil {
			sa.log.Errorf("Syncer unmarshal msg %d err %v", syncMsg.MsgType, err)
			return
		}
		sa.handleBlockRequest(&msg)
	case SyncMessage_BlockResponse:
		var msg BlockResponse
		err := sa.marshaler.Unmarshal(syncMsg.Data, &msg)
		if err != nil {
			sa.log.Errorf("Syncer unmarshal msg %d err %v", syncMsg.MsgType, err)
			return
		}
		sa.handleBlockResponse(&msg)
	case SyncMessage_StateRequest:
		var msg StateRequest
		err := sa.marshaler.Unmarshal(syncMsg.Data, &msg)
		if err != nil {
			sa.log.Errorf("Syncer unmarshal msg %d err %v", syncMsg.MsgType, err)
			return
		}
		sa.handleStateRequest(&msg)
	case SyncMessage_EpochResponse:
		var msg EpochResponse
		err := sa.marshaler.Unmarshal(syncMsg.Data, &msg)
		if err != nil {
			sa.log.Errorf("Syncer unmarshal msg %d err %v", syncMsg.MsgType, err)
			return
		}
		sa.handleEpochResponse(&msg)
	case SyncMessage_NodeStateRequest:
		var msg NodeRequest
		err := sa.marshaler.Unmarshal(syncMsg.Data, &msg)
		if err != nil {
			sa.log.Errorf("Syncer unmarshal msg %d err %v", syncMsg.MsgType, err)
			return
		}
		sa.handleNodeRequest(&msg)
	case SyncMessage_NodeStateResponse:
		var msg NodeResponse
		err := sa.marshaler.Unmarshal(syncMsg.Data, &msg)
		if err != nil {
			sa.log.Errorf("Syncer unmarshal msg %d err %v", syncMsg.MsgType, err)
			return
		}
		sa.handleNodeResponse(&msg)
	default:
		sa.log.Errorf("Syncer receive invalid msg %d", syncMsg.MsgType)
		return
	}
}

func (sync *syncer) Stop() {
	panic("implement me")
}
