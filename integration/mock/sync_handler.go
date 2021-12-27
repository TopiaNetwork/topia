package mock

import (
	"github.com/AsynkronIT/protoactor-go/actor"
	"github.com/TopiaNetwork/topia/codec"
	tplog "github.com/TopiaNetwork/topia/log"
	"github.com/TopiaNetwork/topia/sync"
)

type MockSyncHandler struct {
	Log tplog.Logger
}

func (handler *MockSyncHandler) HandleBlockRequest(context actor.Context, marshaler codec.Marshaler, msg *sync.BlockRequest) error {
	handler.Log.Info("handleBlockRequest")
	bResp := sync.BlockResponse{
		Height: 100,
	}
	data, _ := marshaler.Marshal(&bResp)

	syncMsg := sync.SyncMessage{
		MsgType: sync.SyncMessage_BlockResponse,
		Data:    data,
	}

	syncData, _ := marshaler.Marshal(&syncMsg)

	context.Respond(syncData)

	return nil
}

func (handler *MockSyncHandler) HandleBlockResponse(msg *sync.BlockResponse) error {
	handler.Log.Info("handleBlockRequest")
	return nil
}

func (handler *MockSyncHandler) HandleStatusRequest(context actor.Context, marshaler codec.Marshaler, msg *sync.StatusRequest) error {
	handler.Log.Info("handleBlockRequest")
	sResp := sync.StatusResponse{
		HeightHighest: 145,
		HeightLowest:  10,
	}
	data, _ := marshaler.Marshal(&sResp)

	syncMsg := sync.SyncMessage{
		MsgType: sync.SyncMessage_StatusResponse,
		Data:    data,
	}

	syncData, _ := marshaler.Marshal(&syncMsg)

	context.Respond(syncData)

	return nil
}

func (handler *MockSyncHandler) HandleStatusResponse(msg *sync.StatusResponse) error {
	handler.Log.Info("handleBlockRequest")
	return nil
}
