package sync

import (
	"bytes"
	"context"
	"github.com/TopiaNetwork/topia/chain/types"
	"github.com/TopiaNetwork/topia/common"
	"github.com/TopiaNetwork/topia/ledger"
	"github.com/TopiaNetwork/topia/network/protocol"
	"github.com/TopiaNetwork/topia/service"
	"github.com/TopiaNetwork/topia/state"
	"sync"

	"github.com/TopiaNetwork/topia/codec"
	tplog "github.com/TopiaNetwork/topia/log"
)

const MaxSyncHeights = uint64(86400)

type SyncHandler interface {
	HandleHeartBeatRequest(msg *HeartBeatRequest) error

	HandleBlockRequest(cmsg *BlockRequest) error

	HandleBlockResponse(msg *BlockResponse) error

	HandleStateRequest(cmsg *StateRequest) error

	HandleStateResponse(msg *StateResponse) error

	//
	//HandleEpochRequest(msg *EpochRequest) error
	//
	//HandleEpochResponse(msg *EpochResponse) error
	//
	//HandleNodeRequest(msg *NodeRequest) error
	//
	//HandleNodeResponse(msg *NodeResponse) error

	//HandleAccountRequest(msg *AccountRequest) error
	//
	//HandleAccountResponse(msg *AccountResponse) error
	//
	//HandleChainRequest(msg *ChainRequest) error
	//
	//HandleChainResponse(msg *ChainResponse) error
}

type syncHandler struct {
	log             tplog.Logger
	ctx             context.Context
	marshaler       codec.Marshaler
	ledger          ledger.Ledger
	HighestHeight   uint64
	query           SyncServant
	quitSync        chan struct{}
	blockDownloader *BlockDownloader
	epochDownloader *EpochDownloader
	wg              sync.WaitGroup
}

func NewSyncHandler(log tplog.Logger, ctx context.Context, marshaler codec.Marshaler, ledger ledger.Ledger,
	stateQueryService service.StateQueryService,
	blockService service.BlockService) *syncHandler {
	syhandler := &syncHandler{
		log:       log,
		ctx:       ctx,
		marshaler: marshaler,
		ledger:    ledger,
	}
	syhandler.blockDownloader = NewBlockDownloader(syhandler.log, syhandler.marshaler, syhandler.ctx)
	syhandler.epochDownloader = NewEpochDownloader(syhandler.log, syhandler.marshaler, syhandler.ctx)
	syhandler.query = NewSyncServant(stateQueryService, blockService)
	curHeight, err := syhandler.query.GetLatestHeight()
	if err != nil {
		syhandler.log.Errorf("query current Height Error:", err)
	}
	syhandler.HighestHeight = curHeight

	return syhandler
}

func (sh *syncHandler) HandleHeartBeatRequest(msg *HeartBeatRequest) error {
	curHeight, err := sh.query.GetLatestHeight()
	if err != nil {
		return err
	}
	heights := make([]uint64, 0)
	if curHeight < msg.BlockHeight && sh.HighestHeight < common.MinUint64(curHeight+MaxSyncHeights, msg.BlockHeight) {
		//if localHeight lower than remoteHeight,update local heightHighest ,make Height difference list，
		//send blockRequests to remote node.
		sh.HighestHeight = common.MinUint64(curHeight+MaxSyncHeights, msg.BlockHeight)
		for height := curHeight + 1; height <= msg.BlockHeight; height++ {
			heights = append(heights, height)
			blockRequest := &BlockRequest{
				Height: height,
			}
			data, _ := sh.marshaler.Marshal(blockRequest)
			syncMsg := SyncMessage{
				MsgType: SyncMessage_BlockRequest,
				Data:    data,
			}

			syncData, _ := sh.marshaler.Marshal(&syncMsg)
			sh.query.Send(sh.ctx, protocol.SyncProtocolID_Block, MOD_NAME, syncData)
		}
	}

	curStateRoot, err := sh.query.StateRoot()
	if err != nil {
		return err
	}
	curStateVersion, err := sh.query.StateLatestVersion()
	if err != nil {
		return err
	}
	curEpochRoot, err := sh.query.GetRoundStateRoot()
	if err != nil {
		return err
	}
	curNodeRoot, err := sh.query.GetNodeStateRoot()
	if err != nil {
		return err
	}
	curChainRoot, err := sh.query.GetChainRoot()
	if err != nil {
		return err
	}
	curAccountRoot, err := sh.query.GetAccountRoot()
	if err != nil {
		return err
	}
	if !bytes.Equal(curStateRoot, msg.StateRoot) {
		stateRequest := &StateRequest{
			StateVersion: curStateVersion,
			EpochRoot:    curEpochRoot,
			NodeRoot:     curNodeRoot,
			ChainRoot:    curChainRoot,
			AccountRoot:  curAccountRoot,
		}
		data, _ := sh.marshaler.Marshal(stateRequest)
		syncMsg := SyncMessage{
			MsgType: SyncMessage_StateRequest,
			Data:    data,
		}
		syncData, _ := sh.marshaler.Marshal(&syncMsg)
		sh.query.Send(sh.ctx, protocol.SyncProtocolId_Epoch, MOD_NAME, syncData)

	}
	return nil
}

func (sh *syncHandler) HandleBlockRequest(msg *BlockRequest) error {
	curHeight, err := sh.query.GetLatestHeight()
	if err != nil {
		return err
	}
	if curHeight < msg.Height {
		return nil
	} else {
		requestBlock, err := sh.query.GetBlockByNumber(types.BlockNum(msg.Height))
		if err != nil {
			return err
		}
		blockResponse := &BlockResponse{
			Height: msg.Height,
			Code:   0,
			Block:  requestBlock,
		}

		data, _ := sh.marshaler.Marshal(blockResponse)
		syncMsg := SyncMessage{
			MsgType: SyncMessage_BlockResponse,
			Data:    data,
		}
		syncData, _ := sh.marshaler.Marshal(&syncMsg)
		sh.query.Send(sh.ctx, protocol.SyncProtocolID_Block, MOD_NAME, syncData)
		return nil
	}
}

func (sh *syncHandler) HandleBlockResponse(msg *BlockResponse) error {
	curHeight, err := sh.query.GetLatestHeight()
	if err != nil {
		return err
	}
	if curHeight < msg.Height {
		sh.blockDownloader.chanFetchBlock <- msg.Block
	}
	return nil
}

func (sh *syncHandler) HandleStateRequest(msg *StateRequest) error {
	versions := make([]uint64, 0)
	curStateVersion, err := sh.query.StateLatestVersion()
	if err != nil {
		return err
	}
	curEpochRoot, err := sh.query.GetRoundStateRoot()
	if err != nil {
		return err
	}
	curNodeRoot, err := sh.query.GetNodeStateRoot()
	if err != nil {
		return err
	}
	curChainRoot, err := sh.query.GetChainRoot()
	if err != nil {
		return err
	}
	curAccountRoot, err := sh.query.GetAccountRoot()
	if err != nil {
		return err
	}

	if curStateVersion > msg.StateVersion || msg.EpochRoot == nil {
		//response the state for msg.StateVersion
		stateByVersion := state.CreateCompositionStateReadonlyAt(sh.log, sh.ledger, msg.StateVersion)
		tmpRoot, err := stateByVersion.StateRoot()
		if err != nil {
			return err
		}
		tmpVersion, err := stateByVersion.StateLatestVersion()
		if err != nil {
			return err
		}
		stateDate, _ := sh.marshaler.Marshal(stateByVersion)
		tmpEpochInfo, err := stateByVersion.GetLatestEpoch()
		if err != nil {
			return err
		}
		epochData, _ := sh.marshaler.Marshal(tmpEpochInfo)
		stateResponse := &StateResponse{
			StateRoot:      tmpRoot,
			StateVersion:   tmpVersion,
			StateByVersion: stateDate,
			EpochInfo:      epochData,
		}
		data, _ := sh.marshaler.Marshal(stateResponse)
		syncMsg := SyncMessage{
			MsgType: SyncMessage_StateResponse,
			Data:    data,
		}
		syncData, _ := sh.marshaler.Marshal(&syncMsg)
		sh.query.Send(sh.ctx, protocol.SyncProtocolID_State, MOD_NAME, syncData)

	} else if curStateVersion < msg.StateVersion {
		//resquest the state for versions
		for version := curStateVersion + 1; version <= msg.StateVersion; version++ {
			versions = append(versions, version)
			stateRequest := &StateRequest{
				StateVersion: version,
			}
			data, _ := sh.marshaler.Marshal(stateRequest)
			syncMsg := SyncMessage{
				MsgType: SyncMessage_StateRequest,
				Data:    data,
			}
			syncData, _ := sh.marshaler.Marshal(&syncMsg)
			sh.query.Send(sh.ctx, protocol.SyncProtocolID_State, MOD_NAME, syncData)
		}
	} else if curStateVersion == msg.StateVersion {
		if !bytes.Equal(curEpochRoot, msg.EpochRoot) {
			curEpochInfo, err := sh.query.GetLatestEpoch()
			if err != nil {
				return err
			}
			epochResponse := &EpochResponse{
				Epoch:          curEpochInfo.Epoch,
				StartTimeStamp: curEpochInfo.StartTimeStamp,
				StartHeight:    curEpochInfo.StartHeight,
			}
			data, _ := sh.marshaler.Marshal(epochResponse)
			syncMsg := SyncMessage{
				MsgType: SyncMessage_EpochResponse,
				Data:    data,
			}
			syncData, _ := sh.marshaler.Marshal(&syncMsg)
			sh.query.Send(sh.ctx, protocol.SyncProtocolId_Epoch, MOD_NAME, syncData)
		}
		if !bytes.Equal(curNodeRoot, msg.NodeRoot) {
			tmpState := state.CreateCompositionStateReadonlyAt(sh.log, sh.ledger, msg.StateVersion)
			data, _ := sh.marshaler.Marshal(tmpState)
			nodeResponse := &NodeResponse{
				StateVersion: curStateVersion,
				NodeRoot:     curNodeRoot,
				StateData:    data,
			}
			dataResponse, _ := sh.marshaler.Marshal(nodeResponse)
			syncMsg := SyncMessage{
				MsgType: SyncMessage_NodeStateResponse,
				Data:    dataResponse,
			}
			syncData, _ := sh.marshaler.Marshal(&syncMsg)
			sh.query.Send(sh.ctx, protocol.SyncProtocolId_Node, MOD_NAME, syncData)

		}
		if !bytes.Equal(curChainRoot, msg.ChainRoot) {
			tmpState := state.CreateCompositionStateReadonlyAt(sh.log, sh.ledger, msg.StateVersion)
			data, _ := sh.marshaler.Marshal(tmpState)
			chainResponse := &ChainResponse{
				StateVersion: curStateVersion,
				ChainRoot:    curChainRoot,
				StateData:    data,
			}
			dataResponse, _ := sh.marshaler.Marshal(chainResponse)
			syncMsg := SyncMessage{
				MsgType: SyncMessage_ChainStateResponse,
				Data:    dataResponse,
			}
			syncData, _ := sh.marshaler.Marshal(&syncMsg)
			sh.query.Send(sh.ctx, protocol.SyncProtocolId_Chain, MOD_NAME, syncData)
		}
		if !bytes.Equal(curAccountRoot, msg.AccountRoot) {
			tmpState := state.CreateCompositionStateReadonlyAt(sh.log, sh.ledger, msg.StateVersion)
			data, _ := sh.marshaler.Marshal(tmpState)
			AccountResponse := &AccountResponse{
				StateVersion: curStateVersion,
				AccountRoot:  curAccountRoot,
				StateData:    data,
			}
			dataResponse, _ := sh.marshaler.Marshal(chainResponse)
			syncMsg := SyncMessage{
				MsgType: SyncMessage_ChainStateResponse,
				Data:    dataResponse,
			}
			syncData, _ := sh.marshaler.Marshal(&syncMsg)
			sh.query.Send(sh.ctx, protocol.SyncProtocolId_Chain, MOD_NAME, syncData)
		}
	}

	return nil
}

func (sh *syncHandler) HandleStateResponse(msg *StateResponse) error {

}

//
//func (sh *syncHandler) HandleEpochRequest(msg *EpochRequest) error {
//	curEpochInfo, err := sh.query.GetLatestEpoch()
//	if err != nil {
//		return err
//	}
//	curEpoch := curEpochInfo.Epoch
//	if curEpoch <= msg.Epoch {
//		return nil
//	} else {
//		epochResponse := &EpochResponse{
//			Epoch:          curEpochInfo.Epoch,
//			StartTimeStamp: curEpochInfo.StartTimeStamp,
//			StartHeight:    curEpochInfo.StartHeight,
//		}
//		data, err := sh.Marshaler.Marshal(epochResponse)
//		if err != nil {
//			return err
//		}
//		syncMsg := SyncMessage{
//			MsgType: SyncMessage_BlockRequest,
//			Data:    data,
//		}
//		syncData, _ := sh.Marshaler.Marshal(&syncMsg)
//		sh.query.Send(sh.ctx, protocol.SyncProtocolId_Epoch, MOD_NAME, syncData)
//
//	}
//	return nil
//}

func (sh *syncHandler) HandleEpochResponse(msg *EpochResponse) error {
	curEponchInfo, err := sh.query.GetLatestEpoch()
	if err != nil {
		return err
	}
	curEponch := curEponchInfo.Epoch
	if curEponch < msg.Epoch {
		return nil
	} else {
		remoteEpoch := &common.EpochInfo{
			Epoch:          msg.Epoch,
			StartTimeStamp: msg.StartTimeStamp,
			StartHeight:    msg.StartHeight,
		}
		sh.epochDownloader.chanFetchEpochInfo <- remoteEpoch
	}
	return nil
}

//func (sh *syncHandler) HandleNodeRequest(msg *NodeRequest) error {
//	curNodeStateVersion, err := sh.query.GetNodeLatestStateVersion()
//	if err != nil {
//		return err
//	}
//	if curNodeStateVersion <= msg.StateVersion {
//		return nil
//	} else {
//		nodeState := sh.query.stat
//
//		nodeStateResponse := &NodeResponse{
//			StateVersion:         msg.StateVersion,
//			NodeState:            ,
//			XXX_NoUnkeyedLiteral: struct{}{},
//			XXX_unrecognized:     nil,
//			XXX_sizecache:        0,
//		}
//		data, err := sh.Marshaler.Marshal(epochResponse)
//		if err != nil {
//			return err
//		}
//		syncMsg := SyncMessage{
//			MsgType: SyncMessage_BlockRequest,
//			Data:    data,
//		}
//		syncData, _ := sh.Marshaler.Marshal(&syncMsg)
//		sh.query.Send(sh.ctx, protocol.SyncProtocolId_Epoch, MOD_NAME, syncData)
//
//	}
//	return nil
//}
//
//func (sh *syncHandler) HandleNodeResponse(msg *EpochResponse) error {
//	curEponchInfo, err := sh.query.GetLatestEpoch()
//	if err != nil {
//		return err
//	}
//	curEponch := curEponchInfo.Epoch
//	if curEponch <= msg.Epoch {
//		return nil
//	} else {
//		remoteEpoch := &common.EpochInfo{
//			Epoch:          msg.Epoch,
//			StartTimeStamp: msg.StartTimeStamp,
//			StartHeight:    msg.StartHeight,
//		}
//		sh.epochDownloader.chanFetchEpochInfo <- remoteEpoch
//	}
//	return nil
//}
