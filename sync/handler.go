package sync

import (
	"github.com/AsynkronIT/protoactor-go/actor"

	"github.com/TopiaNetwork/topia/codec"
	tplog "github.com/TopiaNetwork/topia/log"
)

type SyncHandler interface {
	HandleBlockRequest(context actor.Context, marshaler codec.Marshaler, msg *BlockRequest) error

	HandleBlockResponse(msg *BlockResponse) error

	HandleStatusRequest(context actor.Context, marshaler codec.Marshaler, msg *StatusRequest) error

	HandleStatusResponse(msg *StatusResponse) error
}

type syncHandler struct {
	log tplog.Logger
}

func NewSyncHandler(log tplog.Logger) *syncHandler {
	return &syncHandler{
		log: log,
	}
}

func (sa *syncHandler) HandleBlockRequest(context actor.Context, marshaler codec.Marshaler, msg *BlockRequest) error {
	panic("implement me")
}

func (sa *syncHandler) HandleBlockResponse(msg *BlockResponse) error {
	panic("implement me")
}

func (sa *syncHandler) HandleStatusRequest(context actor.Context, marshaler codec.Marshaler, msg *StatusRequest) error {
	panic("implement me")
}

func (sa *syncHandler) HandleStatusResponse(msg *StatusResponse) error {
	panic("implement me")
}
