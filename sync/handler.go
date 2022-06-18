package sync

import (
	"bytes"
	"context"
	"encoding/hex"
	"errors"
	"sync"
	"time"

	"github.com/TopiaNetwork/topia/account"
	"github.com/TopiaNetwork/topia/chain/types"
	tpchaintypes "github.com/TopiaNetwork/topia/chain/types"
	"github.com/TopiaNetwork/topia/codec"
	"github.com/TopiaNetwork/topia/common"
	tpcrtypes "github.com/TopiaNetwork/topia/crypt/types"
	"github.com/TopiaNetwork/topia/ledger"
	tplog "github.com/TopiaNetwork/topia/log"
	"github.com/TopiaNetwork/topia/network/protocol"
	"github.com/TopiaNetwork/topia/service"
	"github.com/TopiaNetwork/topia/state"
)

const (
	MaxSyncHeights   = uint64(86400)
	SendNodeLimit    = 10 * 1024
	SendAccountLimit = 10 * 1024
)

var ErrOnSyncing = errors.New("syncing not done")

type SyncHandler interface {
	HandleHeartBeatRequest(msg *HeartBeatRequest) error
	HandleBlockRequest(msg *BlockRequest) error
	HandleBlockResponse(msg *BlockResponse) error
	HandleStateRequest(msg *StateRequest) error
	HandleEpochResponse(msg *EpochResponse) error
	HandleNodeIDsRequest(msg *NodeIDsRequest) error
	HandleNodeRequest(msg *NodeRequest) error
	HandleNodeResponse(msg *NodeResponse) error
	HandleChainRequest(msg *ChainRequest) error
	HandleChainResponse(msg *ChainResponse) error
	HandleAccountRequest(msg *AccountRequest) error
	HandleAccountAddrsRequest(msg *AccountAddrsRequest) error
	HandleAccountResponse(msg *AccountResponse) error
}

type syncHandler struct {
	log       tplog.Logger
	ctx       context.Context
	marshaler codec.Marshaler
	ledger    ledger.Ledger
	nodeID    string

	HighestHeight     uint64
	query             SyncServant
	forceSync         chan struct{}
	quitSync          chan struct{}
	doneChan          chan error
	blockDownloader   *BlockDownloader
	epochDownloader   *EpochDownloader
	nodeDownloader    *NodeDownloader
	chainDownloader   *ChainDownloader
	accountDownloader *AccountDownloader
	wg                sync.WaitGroup
}

func NewSyncHandler(conf *SyncConfig, log tplog.Logger, ctx context.Context, marshaler codec.Marshaler, ledger ledger.Ledger, nodeID string,
	stateQueryService service.StateQueryService,
	blockService service.BlockService) *syncHandler {
	syhandler := &syncHandler{
		log:       log,
		ctx:       ctx,
		marshaler: marshaler,
		ledger:    ledger,
		nodeID:    nodeID,
	}
	pubSubScores := syhandler.query.PubSubScores()
	for _, pubSubScore := range pubSubScores {
		tmpNodeScore := &nodeScore{
			score:    pubSubScore.Score.Score,
			lastTime: time.Now(),
		}
		syhandler.query.SetNodeScore(pubSubScore.ID, tmpNodeScore)
	}
	syhandler.blockDownloader = NewBlockDownloader(conf, syhandler.log, marshaler, syhandler.ctx, syhandler.ledger, nodeID)
	syhandler.epochDownloader = NewEpochDownloader(conf, syhandler.log, syhandler.marshaler, syhandler.ctx, syhandler.ledger, nodeID)
	syhandler.chainDownloader = NewChainDownloader(conf, syhandler.log, syhandler.marshaler, syhandler.ctx, syhandler.ledger, nodeID)
	syhandler.nodeDownloader = NewNodeDownloader(conf, syhandler.log, syhandler.marshaler, syhandler.ctx, syhandler.ledger)
	syhandler.accountDownloader = NewAccountDownloader(conf, syhandler.log, syhandler.marshaler, syhandler.ctx, syhandler.ledger)

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
				NodeID: []byte(sh.nodeID),
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
	curEpochRoot, err := sh.query.GetEpochRoot()
	if err != nil {
		return err
	}
	curNodeRoot, err := sh.query.GetNodeRoot()
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
			NodeID:       []byte(sh.nodeID),
			StateVersion: curStateVersion,
			EpochRoot:    curEpochRoot,
			NodeRoot:     curNodeRoot,
			ChainRoot:    curChainRoot,
			AccountRoot:  curAccountRoot,
		}
		data, _ := sh.marshaler.Marshal(stateRequest)
		zipData, _ := CompressBytes(data, GZIP)
		syncMsg := SyncMessage{
			MsgType: SyncMessage_StateRequest,
			Data:    zipData,
		}
		syncData, _ := sh.marshaler.Marshal(&syncMsg)
		sh.query.Send(sh.ctx, protocol.AsyncSendProtocolID, MOD_NAME, syncData)

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
			NodeID: []byte(sh.nodeID),
			Height: msg.Height,
			Code:   0,
			Block:  requestBlock,
		}

		data, _ := sh.marshaler.Marshal(blockResponse)
		zipData, _ := CompressBytes(data, GZIP)
		if err != nil {
			return err
		}
		syncMsg := SyncMessage{
			MsgType: SyncMessage_BlockResponse,
			Data:    zipData,
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
		blockHash := tpchaintypes.BlockHash(hex.EncodeToString(msg.BlockHash))
		if !sh.query.IsBadBlockExist(blockHash) {
			score := sh.query.GetNodeScore(string(msg.NodeID))
			item := &FreeItem{
				score: score,
				value: msg.Block,
			}
			sh.blockDownloader.chanFetchBlock <- item
		}
	}
	return nil
}

func (sh *syncHandler) HandleStateRequest(msg *StateRequest) error {
	curStateRoot, err := sh.query.StateRoot()
	if err != nil {
		return err
	}
	curStateVersion, err := sh.query.StateLatestVersion()
	if err != nil {
		return err
	}
	curEpochRoot, err := sh.query.GetEpochRoot()
	if err != nil {
		return err
	}
	curNodeRoot, err := sh.query.GetNodeRoot()
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

	if curStateVersion > msg.StateVersion {
		//response the state for msg.StateVersion
		tmpState := state.CreateCompositionStateReadonlyAt(sh.log, sh.ledger, msg.StateVersion)
		tmpRoot, err := tmpState.StateRoot()
		if err != nil {
			return err
		}
		tmpVersion, err := tmpState.StateLatestVersion()
		if err != nil {
			return err
		}
		err = sh.epochResponseToRequest(tmpState, tmpRoot, tmpVersion)
		if err != nil {
			return err
		}

		err = sh.chainResponseToRequest(tmpState, tmpRoot, tmpVersion)
		if err != nil {
			return err
		}
		err = sh.accountAddrsRequestToStateRequest(tmpState, tmpRoot, tmpVersion)
		if err != nil {
			return err
		}
		err = sh.nodeIDsResponseToStateRequest(tmpState, tmpRoot, tmpVersion)
		if err != nil {
			return err
		}
	} else if curStateVersion < msg.StateVersion {
		//resquest the state for versions
		sh.epochDownloader.DoneSync <- ErrOnSyncing
		sh.chainDownloader.DoneSync <- ErrOnSyncing
		sh.nodeDownloader.DoneSync <- ErrOnSyncing
		sh.accountDownloader.DoneSync <- ErrOnSyncing

		for version := curStateVersion; version <= msg.StateVersion; version++ {
			stateRequest := &StateRequest{
				NodeID:       []byte(sh.nodeID),
				StateVersion: version,
			}
			data, _ := sh.marshaler.Marshal(stateRequest)
			zipData, _ := CompressBytes(data, GZIP)
			syncMsg := SyncMessage{
				MsgType: SyncMessage_StateRequest,
				Data:    zipData,
			}
			syncData, _ := sh.marshaler.Marshal(&syncMsg)
			sh.query.Send(sh.ctx, protocol.AsyncSendProtocolID, MOD_NAME, syncData)
		}
	} else if curStateVersion == msg.StateVersion {
		if !bytes.Equal(curEpochRoot, msg.EpochRoot) {
			sh.epochDownloader.DoneSync <- ErrOnSyncing
			curEpochInfo, err := sh.query.GetLatestEpoch()
			if err != nil {
				return err
			}
			epochResponse := &EpochResponse{
				NodeID:         []byte(sh.nodeID),
				StateRoot:      curStateRoot,
				StateVersion:   curStateVersion,
				EpochRoot:      curEpochRoot,
				Epoch:          curEpochInfo.Epoch,
				StartTimeStamp: curEpochInfo.StartTimeStamp,
				StartHeight:    curEpochInfo.StartHeight,
			}
			dataEpoch, _ := sh.marshaler.Marshal(epochResponse)
			syncMsgEpoch := SyncMessage{
				MsgType: SyncMessage_EpochResponse,
				Data:    dataEpoch,
			}
			syncDataEpoch, _ := sh.marshaler.Marshal(&syncMsgEpoch)
			sh.query.Send(sh.ctx, protocol.AsyncSendProtocolID, MOD_NAME, syncDataEpoch)

		}
		if !bytes.Equal(curNodeRoot, msg.NodeRoot) {
			sh.nodeDownloader.DoneSync <- ErrOnSyncing
			tmpState := state.CreateCompositionStateReadonlyAt(sh.log, sh.ledger, msg.StateVersion)
			tmpRoot, err := tmpState.StateRoot()
			if err != nil {
				return err
			}
			tmpVersion, err := tmpState.StateLatestVersion()
			if err != nil {
				return err
			}
			err = sh.nodeIDsResponseToStateRequest(tmpState, tmpRoot, tmpVersion)
			if err != nil {
				return err
			}
		}
		if !bytes.Equal(curChainRoot, msg.ChainRoot) {
			sh.chainDownloader.DoneSync <- ErrOnSyncing
			tmpState := state.CreateCompositionStateReadonlyAt(sh.log, sh.ledger, msg.StateVersion)
			tmpRoot, err := tmpState.StateRoot()
			if err != nil {
				return err
			}
			tmpVersion, err := tmpState.StateLatestVersion()
			if err != nil {
				return err
			}
			err = sh.chainResponseToRequest(tmpState, tmpRoot, tmpVersion)
			if err != nil {
				return err
			}

		}
		if !bytes.Equal(curAccountRoot, msg.AccountRoot) {
			sh.accountDownloader.DoneSync <- ErrOnSyncing
			tmpState := state.CreateCompositionStateReadonlyAt(sh.log, sh.ledger, msg.StateVersion)
			tmpRoot, err := tmpState.StateRoot()
			if err != nil {
				return err
			}
			tmpVersion, err := tmpState.StateLatestVersion()
			if err != nil {
				return err
			}
			err = sh.accountAddrsRequestToStateRequest(tmpState, tmpRoot, tmpVersion)

			if err != nil {
				return err
			}

		}
	}
	return nil
}

func (sh *syncHandler) HandleEpochResponse(msg *EpochResponse) error {
	curEpochRoot, err := sh.query.GetEpochRoot()
	if err != nil {
		return err
	}
	curStateVersion, err := sh.query.StateLatestVersion()
	if err != nil {
		return err
	}
	curEpochInfo, err := sh.query.GetLatestEpoch()
	if err != nil {
		return err
	}
	curEponch := curEpochInfo.Epoch
	if msg.StateVersion == curStateVersion {
		if bytes.Equal(curEpochRoot, msg.EpochRoot) {
			return nil
		}
		if msg.Epoch > curEponch {
			score := sh.query.GetNodeScore(string(msg.NodeID))
			item := &FreeItem{
				score: score,
				value: msg,
			}
			sh.epochDownloader.chanFetchEpochInfo <- item
			if sh.epochDownloader.DoneSync == nil {
				sh.epochDownloader.DoneSync <- ErrOnSyncing
			}
			sh.epochDownloader.remoteEpochRoot = msg.EpochRoot
		} else if msg.Epoch < curEponch {
			curStateRoot, err := sh.query.StateRoot()
			if err != nil {
				return err
			}
			epochResponse := &EpochResponse{
				NodeID:         []byte(sh.nodeID),
				StateRoot:      curStateRoot,
				StateVersion:   curStateVersion,
				EpochRoot:      curEpochRoot,
				Epoch:          curEponch,
				StartTimeStamp: curEpochInfo.StartTimeStamp,
				StartHeight:    curEpochInfo.StartHeight,
			}
			dataEpoch, _ := sh.marshaler.Marshal(epochResponse)
			syncMsgEpoch := SyncMessage{
				MsgType: SyncMessage_EpochResponse,
				Data:    dataEpoch,
			}
			syncDataEpoch, _ := sh.marshaler.Marshal(&syncMsgEpoch)
			sh.query.Send(sh.ctx, protocol.AsyncSendProtocolID, MOD_NAME, syncDataEpoch)
		}
	} else if msg.StateVersion > curStateVersion {
		score := sh.query.GetNodeScore(string(msg.NodeID))
		item := &FreeItem{
			score: score,
			value: msg,
		}
		sh.epochDownloader.chanFetchEpochInfo <- item
		if sh.nodeDownloader.DoneSync == nil {
			sh.nodeDownloader.DoneSync <- ErrOnSyncing
		}
	}
	return nil
}

func (sh *syncHandler) HandleNodeIDsRequest(msg *NodeIDsRequest) error {
	curRoot, err := sh.query.StateRoot()
	if err != nil {
		return err
	}
	curVersion, err := sh.query.StateLatestVersion()
	if err != nil {
		return err
	}
	curNodeRoot, err := sh.query.GetNodeRoot()
	if err != nil {
		return err
	}
	var msgIDs []string
	err = sh.marshaler.Unmarshal(msg.NodeIDsData, msgIDs)
	if err != nil {
		return err
	}
	if msg.NodeIDsData == nil {
		tmpState := state.CreateCompositionStateReadonlyAt(sh.log, sh.ledger, msg.StateVersion)
		tmpStateRoot, err := tmpState.StateRoot()
		if err != nil {
			return err
		}
		err = sh.nodeIDsResponseToStateRequest(tmpState, tmpStateRoot, msg.StateVersion)
		if err != nil {
			return err
		}

	}
	if curVersion == msg.StateVersion {
		if bytes.Equal(curNodeRoot, msg.NodeRoot) {
			return nil
		}

		tmpState := state.CreateCompositionStateReadonlyAt(sh.log, sh.ledger, msg.StateVersion)

		tmpIDs, err := tmpState.GetAllConsensusNodeIDs()
		if err != nil {
			return err
		}
		var diffIDs []string
		if sh.nodeDownloader.resetCurLevel {
			diffIDs = msgIDs
		} else {
			diffIDs = ListAMinusListBStrings(msgIDs, tmpIDs)
		}

		if len(diffIDs) > 0 {
			if sh.nodeDownloader.DoneSync == nil {
				sh.nodeDownloader.DoneSync <- ErrOnSyncing
			}
			diffIDsData, _ := sh.marshaler.Marshal(diffIDs)
			nodeRequest := &NodeRequest{
				NodeID:       []byte(sh.nodeID),
				StateRoot:    curRoot,
				StateVersion: curVersion,
				NodeRoot:     curNodeRoot,
				NodeIDsData:  diffIDsData,
			}
			data, _ := sh.marshaler.Marshal(nodeRequest)
			zipData, _ := CompressBytes(data, GZIP)
			syncMsgNode := SyncMessage{
				MsgType: SyncMessage_NodeRequest,
				Data:    zipData,
			}
			syncDataIDs, _ := sh.marshaler.Marshal(&syncMsgNode)
			sh.query.Send(sh.ctx, protocol.AsyncSendProtocolID, MOD_NAME, syncDataIDs)

		}

	}
	return nil
}

func (sh *syncHandler) HandleNodeRequest(msg *NodeRequest) error {
	curStateVersion, err := sh.query.StateLatestVersion()
	if err != nil {
		return err
	}
	if curStateVersion >= msg.GetStateVersion() {
		tmpState := state.CreateCompositionStateReadonlyAt(sh.log, sh.ledger, msg.StateVersion)
		tmpRoot, err := tmpState.StateRoot()
		if err != nil {
			return err
		}
		tmpNodeRoot, err := tmpState.GetNodeRoot()
		if err != nil {
			return err
		}
		var nodeIDs []string
		err = sh.marshaler.Unmarshal(msg.NodeIDsData, nodeIDs)
		if err != nil {
			return err
		}
		var nodes []*common.NodeInfo
		for _, ID := range nodeIDs {
			if tmpState.IsNodeExist(ID) {
				node, err := tmpState.GetNode(ID)
				if err != nil {
					return err
				}
				nodes = append(nodes, node)
			}
		}
		if len(nodes) == 0 {
			return nil
		}
		nodeData, err := sh.marshaler.Marshal(nodes)
		if err != nil {
			return err
		}
		nodeResponse := &NodeResponse{
			NodeID:       []byte(sh.nodeID),
			StateRoot:    tmpRoot,
			StateVersion: msg.StateVersion,
			NodeRoot:     tmpNodeRoot,
			NodesData:    nodeData,
		}
		data, _ := sh.marshaler.Marshal(nodeResponse)
		zipData, _ := CompressBytes(data, GZIP)
		syncMsgNode := SyncMessage{
			MsgType: SyncMessage_NodeResponse,
			Data:    zipData,
		}
		syncDataNode, _ := sh.marshaler.Marshal(syncMsgNode)
		sh.query.Send(sh.ctx, protocol.AsyncSendProtocolID, MOD_NAME, syncDataNode)
	}
	return nil
}

func (sh *syncHandler) HandleNodeResponse(msg *NodeResponse) error {
	curStateVersion, err := sh.query.StateLatestVersion()
	if err != nil {
		return err
	}

	if curStateVersion > msg.StateVersion {
		return nil
	} else {
		sh.nodeDownloader.chanFetchNodeFragment <- msg
		sh.nodeDownloader.DoneSync = make(chan error, 1)
		if curStateVersion == msg.StateVersion {
			sh.nodeDownloader.remoteNodeRoot = msg.NodeRoot
		}
	}
	return nil
}

func (sh *syncHandler) HandleChainRequest(msg *ChainRequest) error {
	curStateVersion, err := sh.query.StateLatestVersion()
	if err != nil {
		return err
	}
	if curStateVersion >= msg.GetStateVersion() {
		tmpState := state.CreateCompositionStateReadonlyAt(sh.log, sh.ledger, msg.StateVersion)
		tmpRoot, err := tmpState.StateRoot()
		if err != nil {
			return err
		}
		tmpVersion, err := tmpState.StateLatestVersion()
		if err != nil {
			return err
		}
		err = sh.chainResponseToRequest(tmpState, tmpRoot, tmpVersion)
		if err != nil {
			return err
		}
	}
	return nil
}

func (sh *syncHandler) HandleChainResponse(msg *ChainResponse) error {
	curChainRoot, err := sh.query.GetChainRoot()
	if err != nil {
		return err
	}
	curStateVersion, err := sh.query.StateLatestVersion()
	if err != nil {
		return err
	}

	if curStateVersion > msg.StateVersion {
		return nil
	} else if curStateVersion == msg.StateVersion {
		if bytes.Equal(curChainRoot, msg.ChainRoot) {
			return nil
		}
		curBlockHeight, err := sh.query.GetLatestHeight()
		if err != nil {
			return err
		}
		if curBlockHeight < msg.LatestHeight {
			score := sh.query.GetNodeScore(string(msg.NodeID))
			item := &FreeItem{
				score: score,
				value: msg,
			}
			sh.chainDownloader.chanFetchChainData <- item
			if sh.chainDownloader.DoneSync == nil {
				sh.chainDownloader.DoneSync <- ErrOnSyncing
			}
			sh.chainDownloader.remoteChainRoot = msg.ChainRoot
		} else if curBlockHeight == msg.LatestHeight {
			var msgBlock *tpchaintypes.Block
			err := sh.marshaler.Unmarshal(msg.LatestBlock, &msgBlock)
			if err != nil {
				return err
			}
			msgBlockVersion := msgBlock.Head.Version
			curBlock, err := sh.query.GetLatestBlock()
			if err != nil {
				return err
			}
			curBlockVersion := curBlock.Head.Version
			if curBlockVersion < msgBlockVersion {
				score := sh.query.GetNodeScore(string(msg.NodeID))
				item := &FreeItem{
					score: score,
					value: msg,
				}
				sh.chainDownloader.chanFetchChainData <- item
				if sh.chainDownloader.DoneSync == nil {
					sh.chainDownloader.DoneSync <- ErrOnSyncing
				}
				sh.chainDownloader.remoteChainRoot = msg.ChainRoot
			}
		}

	} else if curStateVersion > msg.StateVersion {
		tmpState := state.CreateCompositionStateReadonlyAt(sh.log, sh.ledger, msg.StateVersion)
		tmpRoot, err := tmpState.StateRoot()
		if err != nil {
			return err
		}
		sh.chainResponseToRequest(tmpState, tmpRoot, msg.StateVersion)
	} else if curStateVersion > msg.StateVersion {
		if sh.chainDownloader.DoneSync == nil {
			sh.chainDownloader.DoneSync <- ErrOnSyncing
		}
		score := sh.query.GetNodeScore(string(msg.NodeID))
		item := &FreeItem{
			score: score,
			value: msg,
		}
		sh.chainDownloader.chanFetchChainData <- item
	}

	return nil
}

func (sh *syncHandler) HandleAccountAddrsRequest(msg *AccountAddrsRequest) error {
	curRoot, err := sh.query.StateRoot()
	if err != nil {
		return err
	}
	curVersion, err := sh.query.StateLatestVersion()
	if err != nil {
		return err
	}
	curAccountRoot, err := sh.query.GetAccountRoot()
	if err != nil {
		return err
	}

	var msgAddrs []tpcrtypes.Address
	err = sh.marshaler.Unmarshal(msg.AccountAddrsData, msgAddrs)
	if err != nil {
		return err
	}
	if msg.AccountAddrsData == nil {
		tmpState := state.CreateCompositionStateReadonlyAt(sh.log, sh.ledger, msg.StateVersion)
		tmpStateRoot, err := tmpState.StateRoot()
		if err != nil {
			return err
		}
		err = sh.accountAddrsRequestToStateRequest(tmpState, tmpStateRoot, msg.StateVersion)
		if err != nil {
			return err
		}
	}
	if curVersion == msg.StateVersion {
		if bytes.Equal(curAccountRoot, msg.AccountRoot) {
			return nil
		}
		tmpState := state.CreateCompositionStateReadonlyAt(sh.log, sh.ledger, msg.StateVersion)
		curAccounts, err := tmpState.GetAllAccounts()
		if err != nil {
			return err
		}
		var curAddrs []tpcrtypes.Address
		for _, acc := range curAccounts {
			curAddrs = append(curAddrs, acc.Addr)
		}
		var diffAddrs []tpcrtypes.Address
		if sh.nodeDownloader.resetCurLevel {
			diffAddrs = msgAddrs
		} else {
			diffAddrs = ListAMinusListBAddr(msgAddrs, curAddrs)
		}

		if len(diffAddrs) > 0 {
			diffAddrsData, _ := sh.marshaler.Marshal(diffAddrs)
			requestAccounts := &AccountRequest{
				NodeID:           []byte(sh.nodeID),
				StateRoot:        curRoot,
				StateVersion:     curVersion,
				AccountRoot:      curAccountRoot,
				AccountAddrsData: diffAddrsData,
			}
			data, _ := sh.marshaler.Marshal(requestAccounts)
			zipData, _ := CompressBytes(data, GZIP)
			syncMsgAccount := SyncMessage{
				MsgType: SyncMessage_AccountAddrsRequest,
				Data:    zipData,
			}
			syncDataAddrs, _ := sh.marshaler.Marshal(&syncMsgAccount)
			sh.query.Send(sh.ctx, protocol.AsyncSendProtocolID, MOD_NAME, syncDataAddrs)
		}
	}
	return nil
}

func (sh *syncHandler) HandleAccountRequest(msg *AccountRequest) error {
	curStateVersion, err := sh.query.StateLatestVersion()
	if err != nil {
		return err
	}
	if curStateVersion >= msg.GetStateVersion() {
		tmpState := state.CreateCompositionStateReadonlyAt(sh.log, sh.ledger, msg.StateVersion)
		tmpRoot, err := tmpState.StateRoot()
		if err != nil {
			return err
		}
		accountRoot, err := tmpState.GetAccountRoot()
		if err != nil {
			return err
		}
		var msgAddrs []tpcrtypes.Address
		err = sh.marshaler.Unmarshal(msg.AccountAddrsData, &msgAddrs)
		if err != nil {
			return err
		}
		var accounts []*account.Account
		for _, addr := range msgAddrs {
			account, err := tmpState.GetAccount(addr)
			if err == nil {
				accounts = append(accounts, account)
			}
		}
		if len(accounts) == 0 {
			return nil
		}
		accountsData, err := sh.marshaler.Marshal(accounts)
		if err != nil {
			return err
		}
		reponseAccounts := &AccountResponse{
			NodeID:       []byte(sh.nodeID),
			StateRoot:    tmpRoot,
			StateVersion: msg.StateVersion,
			AccountRoot:  accountRoot,
			AccountsData: accountsData,
		}
		data, _ := sh.marshaler.Marshal(reponseAccounts)
		zipData, _ := CompressBytes(data, GZIP)
		syncMsgAccount := SyncMessage{
			MsgType: SyncMessage_AccountResponse,
			Data:    zipData,
		}
		syncDataAccounts, _ := sh.marshaler.Marshal(&syncMsgAccount)
		sh.query.Send(sh.ctx, protocol.AsyncSendProtocolID, MOD_NAME, syncDataAccounts)
	}
	return nil

}

func (sh *syncHandler) HandleAccountResponse(msg *AccountResponse) error {
	curStateVersion, err := sh.query.StateLatestVersion()
	if err != nil {
		return err
	}

	if curStateVersion > msg.StateVersion {
		return nil
	} else {
		sh.accountDownloader.chanFetchAccountFragment <- msg
		sh.accountDownloader.DoneSync = make(chan error, 1)
		if curStateVersion == msg.StateVersion {
			sh.accountDownloader.remoteAccountRoot = msg.AccountRoot
		}
	}
	return nil
}

func (sh *syncHandler) epochResponseToRequest(tmpState state.CompositionStateReadonly, tmpRoot []byte, tmpVersion uint64) error {

	EpochRoot, err := tmpState.GetEpochRoot()
	if err != nil {
		return err
	}

	tmpEpochInfo, err := tmpState.GetLatestEpoch()
	if err != nil {
		return err
	}
	epochResponse := &EpochResponse{
		NodeID:         []byte(sh.nodeID),
		StateRoot:      tmpRoot,
		StateVersion:   tmpVersion,
		EpochRoot:      EpochRoot,
		Epoch:          tmpEpochInfo.Epoch,
		StartTimeStamp: tmpEpochInfo.StartTimeStamp,
		StartHeight:    tmpEpochInfo.StartHeight,
	}
	dataEpoch, _ := sh.marshaler.Marshal(epochResponse)
	syncMsgEpoch := SyncMessage{
		MsgType: SyncMessage_EpochResponse,
		Data:    dataEpoch,
	}
	syncDataEpoch, _ := sh.marshaler.Marshal(&syncMsgEpoch)
	sh.query.Send(sh.ctx, protocol.AsyncSendProtocolID, MOD_NAME, syncDataEpoch)
	return nil
}

func (sh *syncHandler) chainResponseToRequest(tmpState state.CompositionStateReadonly, tmpRoot []byte, tmpVersion uint64) error {
	ChainRoot, err := tmpState.GetChainRoot()
	if err != nil {
		return err
	}
	ChainID := tmpState.ChainID()
	NetworkType := tmpState.NetworkType()
	dataNetworkType, _ := sh.marshaler.Marshal(NetworkType)
	LatestBlock, err := tmpState.GetLatestBlock()
	LatestHeight := LatestBlock.Head.Height
	if err != nil {
		return err
	}
	dataLatestBlock, _ := sh.marshaler.Marshal(LatestBlock)
	LatestBlockResult, err := tmpState.GetLatestBlockResult()
	if err != nil {
		return err
	}
	dataLatestBlockResult, _ := sh.marshaler.Marshal(LatestBlockResult)
	chainResponse := &ChainResponse{
		NodeID:            []byte(sh.nodeID),
		StateRoot:         tmpRoot,
		StateVersion:      tmpVersion,
		ChainRoot:         ChainRoot,
		ChainID:           string(ChainID),
		NetworkType:       dataNetworkType,
		LatestHeight:      LatestHeight,
		LatestBlock:       dataLatestBlock,
		LatestBlockResult: dataLatestBlockResult,
	}
	data, _ := sh.marshaler.Marshal(chainResponse)
	zipData, _ := CompressBytes(data, GZIP)
	syncMsgChain := SyncMessage{
		MsgType: SyncMessage_ChainResponse,
		Data:    zipData,
	}
	syncDataChain, _ := sh.marshaler.Marshal(&syncMsgChain)
	sh.query.Send(sh.ctx, protocol.AsyncSendProtocolID, MOD_NAME, syncDataChain)
	return nil
}

func (sh *syncHandler) accountAddrsRequestToStateRequest(tmpState state.CompositionStateReadonly, tmpRoot []byte, tmpVersion uint64) error {
	AccountRoot, err := tmpState.GetAccountRoot()
	if err != nil {
		return err
	}
	AllAccounts, err := tmpState.GetAllAccounts()
	if err != nil {
		return err
	}
	var allAccountAddrs []tpcrtypes.Address
	for _, account := range AllAccounts {
		allAccountAddrs = append(allAccountAddrs, account.Addr)
	}

	var segAccountAddrs []tpcrtypes.Address
	lenAccountAddrs := uint64(len(allAccountAddrs))
	accountAddrSegments := (lenAccountAddrs-1)/SendAccountLimit + 1
	for i := uint64(0); i < accountAddrSegments; i++ {
		var accountAddrsData []byte
		if i < accountAddrSegments-1 {
			segAccountAddrs = allAccountAddrs[i*SendAccountLimit : (i+1)*SendAccountLimit]
		} else {
			segAccountAddrs = allAccountAddrs[i*SendAccountLimit:]
		}
		accountAddrsData, _ = sh.marshaler.Marshal(segAccountAddrs)
		accountResponse := &AccountAddrsRequest{
			NodeID:           []byte(sh.nodeID),
			StateRoot:        tmpRoot,
			StateVersion:     tmpVersion,
			AccountRoot:      AccountRoot,
			AccountAddrsData: accountAddrsData,
		}
		data, _ := sh.marshaler.Marshal(accountResponse)
		zipData, _ := CompressBytes(data, GZIP)
		syncMsgAccount := SyncMessage{
			MsgType: SyncMessage_AccountResponse,
			Data:    zipData,
		}
		syncDataAccount, _ := sh.marshaler.Marshal(&syncMsgAccount)
		sh.query.Send(sh.ctx, protocol.AsyncSendProtocolID, MOD_NAME, syncDataAccount)
	}
	return nil
}

func (sh *syncHandler) nodeIDsResponseToStateRequest(tmpState state.CompositionStateReadonly, tmpRoot []byte, tmpVersion uint64) error {

	tmpNodeRoot, err := tmpState.GetNodeRoot()
	if err != nil {
		return err
	}
	allNodeIDs, err := tmpState.GetAllConsensusNodeIDs()
	if err != nil {
		return err
	}
	lenNodeIDs := uint64(len(allNodeIDs))
	nodeIDsSegments := (lenNodeIDs-1)/SendNodeLimit + 1
	var segNodeIDs []string
	for i := uint64(0); i < nodeIDsSegments; i++ {
		if i < nodeIDsSegments-1 {
			segNodeIDs = allNodeIDs[i*SendNodeLimit : (i+1)*SendNodeLimit]
		} else {
			segNodeIDs = allNodeIDs[i*SendNodeLimit:]
		}
		nodeIDsData, _ := sh.marshaler.Marshal(segNodeIDs)
		nodeIDsRequest := &NodeIDsRequest{
			NodeID:       []byte(sh.nodeID),
			StateRoot:    tmpRoot,
			StateVersion: tmpVersion,
			NodeRoot:     tmpNodeRoot,
			NodeIDsData:  nodeIDsData,
		}

		data, _ := sh.marshaler.Marshal(nodeIDsRequest)
		zipData, _ := CompressBytes(data, GZIP)
		syncMsgNode := SyncMessage{
			MsgType: SyncMessage_NodeIDsRequest,
			Data:    zipData,
		}
		syncDataNodeIDsRequest, _ := sh.marshaler.Marshal(&syncMsgNode)
		sh.query.Send(sh.ctx, protocol.AsyncSendProtocolID, MOD_NAME, syncDataNodeIDsRequest)
	}
	return nil
}

func ListAMinusListBAddr(a, b []tpcrtypes.Address) []tpcrtypes.Address {
	keep := make([]tpcrtypes.Address, 0)
	remove := make(map[tpcrtypes.Address]struct{})
	for _, value := range b {
		remove[value] = struct{}{}
	}
	for _, v := range a {
		if _, ok := remove[v]; !ok {
			keep = append(keep, v)
		}
	}
	return keep
}
func ListAMinusListBStrings(a, b []string) []string {
	keep := make([]string, 0)
	remove := make(map[string]struct{})
	for _, value := range b {
		remove[value] = struct{}{}
	}
	for _, v := range a {
		if _, ok := remove[v]; !ok {
			keep = append(keep, v)
		}
	}
	return keep
}
