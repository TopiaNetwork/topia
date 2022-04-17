package chain

import (
	"context"
	"github.com/AsynkronIT/protoactor-go/actor"
	"github.com/TopiaNetwork/topia/configuration"
	"github.com/TopiaNetwork/topia/ledger"
	tpnetprotoc "github.com/TopiaNetwork/topia/network/protocol"

	tpchaintypes "github.com/TopiaNetwork/topia/chain/types"
	"github.com/TopiaNetwork/topia/codec"
	tplog "github.com/TopiaNetwork/topia/log"
	tplogcmm "github.com/TopiaNetwork/topia/log/common"
	tpnet "github.com/TopiaNetwork/topia/network"
)

const (
	MOD_NAME       = "chain"
	MOD_ACTOR_NAME = "chain_actor"
)

type Chain interface {
	Start(sysActor *actor.ActorSystem, network tpnet.Network) error

	Stop()
}

type chain struct {
	log           tplog.Logger
	nodeID        string
	level         tplogcmm.LogLevel
	marshaler     codec.Marshaler
	blkSubProcess BlockInfoSubProcessor
	config        *configuration.Configuration
}

func NewChain(level tplogcmm.LogLevel,
	log tplog.Logger,
	nodeID string,
	codecType codec.CodecType,
	ledger ledger.Ledger,
	config *configuration.Configuration) Chain {
	chainLog := tplog.CreateModuleLogger(level, MOD_NAME, log)
	marshaler := codec.CreateMarshaler(codecType)

	blkSubPro := NewBlockInfoSubProcessor(chainLog, nodeID, marshaler, ledger, config)

	return &chain{
		log:           chainLog,
		level:         level,
		marshaler:     marshaler,
		blkSubProcess: blkSubPro,
	}
}

func (c *chain) dispatch(actorCtx actor.Context, data []byte) {
	var pubsubMsgBlk tpchaintypes.PubSubMessageBlockInfo
	err := c.marshaler.Unmarshal(data, &pubsubMsgBlk)
	if err == nil {
		err = c.blkSubProcess.Process(context.Background(), &pubsubMsgBlk)
		if err != nil {
			c.log.Errorf("Processs block info pubsub message err: %v", err)
		}
	} else {
		c.log.Errorf("chain received invalid message and can't Unmarshal: %v", err)
	}
}

func (c *chain) Start(sysActor *actor.ActorSystem, network tpnet.Network) error {
	actorPID, err := CreateChainActor(c.log, sysActor, c)
	if err != nil {
		c.log.Panicf("CreateChainActor error: %v", err)
		return err
	}

	network.RegisterModule(MOD_NAME, actorPID, c.marshaler)

	err = network.Subscribe(context.Background(), tpnetprotoc.PubSubProtocolID_BlockInfo, c.blkSubProcess.Validate)
	if err != nil {
		c.log.Panicf("Chain subscribe block info pubsub err: %v", err)
		return err
	}

	return nil
}

func (c *chain) Stop() {

}
