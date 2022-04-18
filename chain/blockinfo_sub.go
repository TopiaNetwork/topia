package chain

import (
	"context"
	"errors"
	"fmt"
	tpchaintypes "github.com/TopiaNetwork/topia/chain/types"
	tpcmm "github.com/TopiaNetwork/topia/common"
	"github.com/TopiaNetwork/topia/configuration"
	"github.com/TopiaNetwork/topia/eventhub"
	"github.com/TopiaNetwork/topia/ledger"
	"github.com/TopiaNetwork/topia/state"
	"time"

	"github.com/TopiaNetwork/topia/codec"
	tplog "github.com/TopiaNetwork/topia/log"
	tpnetmsg "github.com/TopiaNetwork/topia/network/message"
)

type BlockInfoSubProcessor interface {
	Validate(ctx context.Context, isLocal bool, data []byte) tpnetmsg.ValidationResult
	Process(ctx context.Context, subMsgBlockInfo *tpchaintypes.PubSubMessageBlockInfo) error
}

type blockInfoSubProcessor struct {
	log       tplog.Logger
	nodeID    string
	marshaler codec.Marshaler
	ledger    ledger.Ledger
	config    *configuration.Configuration
}

func NewBlockInfoSubProcessor(log tplog.Logger, nodeID string, marshaler codec.Marshaler, ledger ledger.Ledger, config *configuration.Configuration) BlockInfoSubProcessor {
	return &blockInfoSubProcessor{
		log:       log,
		nodeID:    nodeID,
		marshaler: marshaler,
		ledger:    ledger,
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

func (bsp *blockInfoSubProcessor) Process(ctx context.Context, subMsgBlockInfo *tpchaintypes.PubSubMessageBlockInfo) error {
	block, blockRS, err := bsp.checkAndParseSubData(subMsgBlockInfo)
	if err != nil {
		return err
	}

	bsp.log.Infof("Process pubsub message: height=%d, result status %s", block.Head.Height, blockRS.Head.Status.String())

	csState := state.GetStateBuilder().CreateCompositionState(bsp.log, bsp.nodeID, bsp.ledger, block.Head.Height)
	if csState == nil {
		err = fmt.Errorf("Nil csState and can't process pubsub message: height=%d", block.Head.Height)
		return err
	}

	latestBlock, err := csState.GetLatestBlock()
	if err != nil {
		err = fmt.Errorf("Can't get the latest block: %v, can't process pubsub message: height=%d", err, block.Head.Height)
		return err
	}

	if latestBlock.Head.Height >= block.Head.Height {
		bsp.log.Warnf("Receive delay PubSubMessageBlockInfo: height=%d, latest block height=%d", block.Head.Height, latestBlock.Head.Height)
		return nil
	}

	err = csState.SetLatestBlock(block)
	if err != nil {
		bsp.log.Errorf("Save the latest block error: %v", err)
		return err
	}
	err = csState.SetLatestBlockResult(blockRS)
	if err != nil {
		bsp.log.Errorf("Save the latest block result error: %v", err)
		return err
	}

	latestEpoch, err := csState.GetLatestEpoch()
	if err != nil {
		bsp.log.Errorf("Can't get latest epoch error: %v", err)
		return err
	}

	var newEpoch *tpcmm.EpochInfo
	deltaH := int(block.Head.Height) - int(latestEpoch.StartHeight)
	if deltaH == int(bsp.config.CSConfig.EpochInterval) {
		newEpoch = &tpcmm.EpochInfo{
			Epoch:          latestEpoch.Epoch + 1,
			StartTimeStamp: uint64(time.Now().UnixNano()),
			StartHeight:    block.Head.Height,
		}
		err = csState.SetLatestEpoch(newEpoch)
		if err != nil {
			bsp.log.Errorf("Save the latest epoch error: %v", err)
			return err
		}
	}

	csState.Commit()
	//ToDo Save new block and block result to block store

	csState.UpdataCompSState(state.CompSState_Commited)

	if newEpoch != nil {
		eventhub.GetEventHubManager().GetEventHub(bsp.nodeID).Trig(ctx, eventhub.EventName_EpochNew, newEpoch)
	}

	eventhub.GetEventHubManager().GetEventHub(bsp.nodeID).Trig(ctx, eventhub.EventName_BlockAdded, block)

	return nil
}
