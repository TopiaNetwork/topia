package sync

import (
	"context"
	"errors"
	tpchaintypes "github.com/TopiaNetwork/topia/chain/types"
	"github.com/TopiaNetwork/topia/codec"

	tplog "github.com/TopiaNetwork/topia/log"
	tpnetmsg "github.com/TopiaNetwork/topia/network/message"
)

type BlockInfoSubProcessor interface {
	Validate(ctx context.Context, isLocal bool, data []byte) tpnetmsg.ValidationResult
	Process(subMsgBlockInfo *tpchaintypes.PubSubMessageBlockInfo) error
}

type blockInfoSubProcessor struct {
	log       tplog.Logger
	marshaler codec.Marshaler
}

func NewBlockInfoSubProcessor(log tplog.Logger, marshaler codec.Marshaler) BlockInfoSubProcessor {
	return &blockInfoSubProcessor{
		log:       log,
		marshaler: marshaler,
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
		err := errors.New("Received blank block info pubsub message")
		bsp.log.Errorf("%v", err)
		return tpnetmsg.ValidationReject
	}

	var blockInfo tpchaintypes.PubSubMessageBlockInfo
	err := bsp.marshaler.Unmarshal(data, &blockInfo)
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

func (bsp *blockInfoSubProcessor) Process(subMsgBlockInfo *tpchaintypes.PubSubMessageBlockInfo) error {
	block, blockRS, err := bsp.checkAndParseSubData(subMsgBlockInfo)
	if err != nil {
		return err
	}

	bsp.log.Infof("Process pubsub message: height=%d, result status", block.Head.Height, blockRS.Head.Status.String())

	//todo process froward block and blockRS

	return nil
}
