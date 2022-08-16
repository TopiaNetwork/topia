package chain

import (
	"context"
	"encoding/json"
	"github.com/AsynkronIT/protoactor-go/actor"
	"github.com/TopiaNetwork/topia/configuration"
	"github.com/TopiaNetwork/topia/execution"
	"github.com/TopiaNetwork/topia/ledger"
	tpnetmsg "github.com/TopiaNetwork/topia/network/message"
	tpnetprotoc "github.com/TopiaNetwork/topia/network/protocol"
	txbasic "github.com/TopiaNetwork/topia/transaction/basic"
	txpool "github.com/TopiaNetwork/topia/transaction_pool/interface"

	tpchaintypes "github.com/TopiaNetwork/topia/chain/types"
	"github.com/TopiaNetwork/topia/codec"
	tplog "github.com/TopiaNetwork/topia/log"
	tplogcmm "github.com/TopiaNetwork/topia/log/common"
	tpnet "github.com/TopiaNetwork/topia/network"
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
	txPool        txpool.TransactionPool
	blkSubProcess BlockInfoSubProcessor
	config        *configuration.Configuration
}

func NewChain(level tplogcmm.LogLevel,
	log tplog.Logger,
	nodeID string,
	codecType codec.CodecType,
	ledger ledger.Ledger,
	txPool txpool.TransactionPool,
	scheduler execution.ExecutionScheduler,
	config *configuration.Configuration) Chain {
	chainLog := tplog.CreateModuleLogger(level, tpchaintypes.MOD_NAME, log)
	marshaler := codec.CreateMarshaler(codecType)

	blkSubPro := NewBlockInfoSubProcessor(chainLog, nodeID, marshaler, ledger, scheduler, config)

	return &chain{
		log:           chainLog,
		nodeID:        nodeID,
		level:         level,
		marshaler:     marshaler,
		txPool:        txPool,
		blkSubProcess: blkSubPro,
	}
}

func (c *chain) dispatch(actorCtx actor.Context, data []byte) {
	var chainMsg tpchaintypes.ChainMessage
	err := c.marshaler.Unmarshal(data, &chainMsg)
	if err != nil {
		c.log.Errorf("Chain receive invalid data %v", err)
		return
	}

	switch chainMsg.MsgType {
	case tpchaintypes.ChainMessage_BlockInfo:
		var pubsubMsgBlk tpchaintypes.PubSubMessageBlockInfo
		err = c.marshaler.Unmarshal(chainMsg.Data, &pubsubMsgBlk)
		if err == nil {
			err = c.blkSubProcess.Process(context.Background(), &pubsubMsgBlk)
			if err != nil {
				c.log.Errorf("Process block info pubsub message err: %v", err)
			}
		} else {
			c.log.Errorf("chain received invalid block info message and can't Unmarshal: %v", err)
		}
	case tpchaintypes.ChainMessage_Tx:
		var tx txbasic.Transaction
		err = c.marshaler.Unmarshal(chainMsg.Data, &tx)

		txID := txbasic.TxID("")
		if err == nil {
			txID, _ = tx.TxID()
			err = c.txPool.AddTx(&tx, true)
		} else {
			c.log.Errorf("chain received invalid tx message and can't Unmarshal: txID %s %v", txID, err)
		}

		var respData tpnetmsg.ResponseData
		if err != nil {
			respData.ErrMsg = err.Error()
		}
		respDataBytes, _ := json.Marshal(&respData)

		actorCtx.Respond(&respDataBytes)

		c.log.Infof("Successfully respond tx message: txID %s", txID, err)
	default:
		c.log.Errorf("chain received unknown message: %s", chainMsg.MsgType.String())
	}
}

func (c *chain) Start(sysActor *actor.ActorSystem, network tpnet.Network) error {
	actorPID, err := CreateChainActor(c.log, sysActor, c)
	if err != nil {
		c.log.Panicf("CreateChainActor error: %v", err)
		return err
	}

	network.RegisterModule(tpchaintypes.MOD_NAME, actorPID, c.marshaler)

	err = network.Subscribe(context.Background(), tpnetprotoc.PubSubProtocolID_BlockInfo, true, c.blkSubProcess.Validate)
	if err != nil {
		c.log.Panicf("Chain subscribe block info pubsub err: %v", err)
		return err
	}

	return nil
}

func (c *chain) Stop() {

}
