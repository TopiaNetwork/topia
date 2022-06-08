package sync

import (
	"github.com/AsynkronIT/protoactor-go/actor"
	"github.com/TopiaNetwork/topia/codec"
	tplog "github.com/TopiaNetwork/topia/log"
	tplogcmm "github.com/TopiaNetwork/topia/log/common"
	"github.com/TopiaNetwork/topia/network"
)

const (
	MOD_NAME       = "sync"
	MOD_ACTOR_NAME = "sync_actor"
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
	handler   SyncHandler
	marshaler codec.Marshaler
}

func NewSyncer(level tplogcmm.LogLevel, log tplog.Logger, codecType codec.CodecType) Syncer {
	syncLog := tplog.CreateModuleLogger(level, MOD_NAME, log)

	return &syncer{
		log:       syncLog,
		level:     level,
		handler:   NewSyncHandler(syncLog),
		marshaler: codec.CreateMarshaler(codecType),
	}
}

func (sa *syncer) UpdateHandler(handler SyncHandler) {
	sa.handler = handler
}

func (sa *syncer) Marshaler() codec.Marshaler {
	return sa.marshaler
}

func (sa *syncer) handleBlockRequest(context actor.Context, msg *BlockRequest) error {
	return sa.handler.HandleBlockRequest(context, sa.marshaler, msg)
}

func (sa *syncer) handleBlockResponse(msg *BlockResponse) error {
	return sa.handler.HandleBlockResponse(msg)
}

func (sa *syncer) handleStatusRequest(context actor.Context, msg *StatusRequest) error {
	return sa.handler.HandleStatusRequest(context, sa.marshaler, msg)
}

func (sa *syncer) handleStatusResponse(msg *StatusResponse) error {
	return sa.handler.HandleStatusResponse(msg)
}

func (sync *syncer) Start(sysActor *actor.ActorSystem, network network.Network) error {
	actorPID, err := CreateSyncActor(sync.level, sync.log, sysActor, sync)
	if err != nil {
		sync.log.Panicf("CreateSyncActor error: %v", err)
		return err
	}

	network.RegisterModule(MOD_NAME, actorPID, sync.marshaler)

	return nil
}

func (sync *syncer) dispatch(context actor.Context, data []byte) {
	var syncMsg SyncMessage
	err := sync.marshaler.Unmarshal(data, &syncMsg)
	if err != nil {
		sync.log.Errorf("Syncer receive invalid data %v", data)
		return
	}

	switch syncMsg.MsgType {
	case SyncMessage_BlockRequest:
		var msg BlockRequest
		err := sync.marshaler.Unmarshal(syncMsg.Data, &msg)
		if err != nil {
			sync.log.Errorf("Syncer unmarshal msg %d err %v", syncMsg.MsgType, err)
			return
		}
		sync.handleBlockRequest(context, &msg)
	case SyncMessage_BlockResponse:
		var msg BlockResponse
		err := sync.marshaler.Unmarshal(syncMsg.Data, &msg)
		if err != nil {
			sync.log.Errorf("Syncer unmarshal msg %d err %v", syncMsg.MsgType, err)
			return
		}
		sync.handleBlockResponse(&msg)
	case SyncMessage_StatusRequest:
		var msg StatusRequest
		err := sync.marshaler.Unmarshal(syncMsg.Data, &msg)
		if err != nil {
			sync.log.Errorf("Syncer unmarshal msg %d err %v", syncMsg.MsgType, err)
			return
		}
		sync.handleStatusRequest(context, &msg)
	case SyncMessage_StatusResponse:
		var msg StatusResponse
		err := sync.marshaler.Unmarshal(syncMsg.Data, &msg)
		if err != nil {
			sync.log.Errorf("Syncer unmarshal msg %d err %v", syncMsg.MsgType, err)
			return
		}
		sync.handleStatusResponse(&msg)
	default:
		sync.log.Errorf("Syncer receive invalid msg %d", syncMsg.MsgType)
		return
	}
}

func (sync *syncer) Stop() {
	panic("implement me")
}
