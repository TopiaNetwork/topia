package chain

import (
	"context"
	"errors"
	"fmt"
	"sync"

	tpchaintypes "github.com/TopiaNetwork/topia/chain/types"
	"github.com/TopiaNetwork/topia/codec"
	"github.com/TopiaNetwork/topia/configuration"
	"github.com/TopiaNetwork/topia/execution"
	"github.com/TopiaNetwork/topia/ledger"
	tplog "github.com/TopiaNetwork/topia/log"
	tpnetmsg "github.com/TopiaNetwork/topia/network/message"
	"github.com/TopiaNetwork/topia/state"
)

type BlockInfoSubProcessor interface {
	Validate(ctx context.Context, isLocal bool, data []byte) tpnetmsg.ValidationResult
	Process(ctx context.Context, subMsgBlockInfo *tpchaintypes.PubSubMessageBlockInfo) error
}

type blockInfoSubProcessor struct {
	log         tplog.Logger
	nodeID      string
	cType       state.CompStateBuilderType
	marshaler   codec.Marshaler
	ledger      ledger.Ledger
	scheduler   execution.ExecutionScheduler
	config      *configuration.Configuration
	syncProcess sync.RWMutex
}

func NewBlockInfoSubProcessor(log tplog.Logger, nodeID string, marshaler codec.Marshaler, ledger ledger.Ledger, scheduler execution.ExecutionScheduler, config *configuration.Configuration) BlockInfoSubProcessor {
	csStateRN := state.CreateCompositionStateReadonly(log, ledger)
	defer csStateRN.Stop()

	isExecutor := csStateRN.IsExistActiveExecutor(nodeID)
	cType := state.CompStateBuilderType_Full
	if !isExecutor {
		cType = state.CompStateBuilderType_Simple
	}

	return &blockInfoSubProcessor{
		log:       log,
		nodeID:    nodeID,
		cType:     cType,
		marshaler: marshaler,
		ledger:    ledger,
		scheduler: scheduler,
		config:    config,
	}
}

func (bsp *blockInfoSubProcessor) validateRemoteBlockInfo(block *tpchaintypes.Block, blockResult *tpchaintypes.BlockResult) tpnetmsg.ValidationResult {
	return tpnetmsg.ValidationAccept
}

func (bsp *blockInfoSubProcessor) checkAndParseSubData(subMsgBlockInfo *tpchaintypes.PubSubMessageBlockInfo) (*tpchaintypes.Block, *tpchaintypes.BlockResult, error) {
	if subMsgBlockInfo == nil {
		err := errors.New("Received blank block info pubsub message")
		bsp.log.Errorf("%v", err)
		return nil, nil, err
	}

	var block tpchaintypes.Block
	err := bsp.marshaler.Unmarshal(subMsgBlockInfo.Block, &block)
	if err != nil {
		bsp.log.Errorf("Can't Unmarshal block from received blank block info pubsub message: %v", err)
		return nil, nil, err
	}

	var blockRS tpchaintypes.BlockResult
	err = bsp.marshaler.Unmarshal(subMsgBlockInfo.BlockResult, &blockRS)
	if err != nil {
		bsp.log.Errorf("Can't Unmarshal block from received blank block info pubsub message: %v", err)
		return nil, nil, err
	}

	return &block, &blockRS, nil
}

func (bsp *blockInfoSubProcessor) Validate(ctx context.Context, isLocal bool, data []byte) tpnetmsg.ValidationResult {
	if data == nil {
		err := errors.New("Chain received blank pubsub data")
		bsp.log.Errorf("%v", err)
		return tpnetmsg.ValidationReject
	}

	var chainMsg tpchaintypes.ChainMessage
	err := bsp.marshaler.Unmarshal(data, &chainMsg)
	if err != nil {
		bsp.log.Errorf("Received invalid chain msg: %v", err)
		return tpnetmsg.ValidationReject
	}

	var blockInfo tpchaintypes.PubSubMessageBlockInfo
	err = bsp.marshaler.Unmarshal(chainMsg.Data, &blockInfo)
	if err != nil {
		bsp.log.Errorf("Can't Unmarshal received block info pubsub message: %v", err)
		return tpnetmsg.ValidationReject
	}

	block, blockRS, err := bsp.checkAndParseSubData(&blockInfo)
	if err != nil {
		return tpnetmsg.ValidationReject
	}

	bsp.log.Infof("Received pubsub message: isLocal=%v, height=%d", isLocal, block.Head.Height)

	if isLocal {
		return tpnetmsg.ValidationAccept
	}

	return bsp.validateRemoteBlockInfo(block, blockRS)
}

func (bsp *blockInfoSubProcessor) Process(ctx context.Context, subMsgBlockInfo *tpchaintypes.PubSubMessageBlockInfo) error {
	bsp.syncProcess.Lock()
	defer bsp.syncProcess.Unlock()

	block, blockRS, err := bsp.checkAndParseSubData(subMsgBlockInfo)
	if err != nil {
		return err
	}

	bsp.log.Infof("Process of pubsub message: height=%d, result status %s, self node %s", block.Head.Height, blockRS.Head.Status.String(), bsp.nodeID)

	csStateRN := state.CreateCompositionStateReadonly(bsp.log, bsp.ledger)
	latestBlock, err := csStateRN.GetLatestBlock()
	if err != nil {
		err = fmt.Errorf("Can't get the latest block: %v, can't process pubsub message: height=%d", err, block.Head.Height)
		csStateRN.Stop()
		return err
	}
	csStateRN.Stop()
	if latestBlock.Head.Height >= block.Head.Height {
		bsp.log.Warnf("Receive delay PubSubMessageBlockInfo: height=%d, latest block height=%d, self node %s", block.Head.Height, latestBlock.Head.Height, bsp.nodeID)
		return nil
	}

	bsp.log.Infof("Process of pubsub message begins committing block: height=%d, result status %s, self node %s", block.Head.Height, blockRS.Head.Status.String(), bsp.nodeID)

	err = bsp.scheduler.CommitBlock(ctx, block.Head.Height, block, blockRS, latestBlock, bsp.ledger, bsp.cType, "ChainBlockSubscriber")
	if err != nil {
		bsp.log.Errorf("Chain block subscriber CommitBlock err: %v, height %d, latest block %d, self node %s", err, block.Head.Height, latestBlock.Head.Height, bsp.nodeID)
		return err
	}
	bsp.log.Infof("Process of pubsub message finishes committing block: height=%d, result status %s, self node %s", block.Head.Height, blockRS.Head.Status.String(), bsp.nodeID)

	return nil
}
