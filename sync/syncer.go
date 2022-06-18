package sync

import (
	"context"
	"sync"
	"time"

	"github.com/AsynkronIT/protoactor-go/actor"
	"github.com/TopiaNetwork/topia/codec"
	"github.com/TopiaNetwork/topia/crypt/types"
	"github.com/TopiaNetwork/topia/ledger"
	tplog "github.com/TopiaNetwork/topia/log"
	tplogcmm "github.com/TopiaNetwork/topia/log/common"
	"github.com/TopiaNetwork/topia/network"
	tpnetcmn "github.com/TopiaNetwork/topia/network/common"
	tpnetmsg "github.com/TopiaNetwork/topia/network/message"
	"github.com/TopiaNetwork/topia/network/protocol"
	"github.com/TopiaNetwork/topia/service"
)

const (
	MOD_NAME                  = "sync"
	MOD_ACTOR_NAME            = "sync_actor"
	NodeScoreUpdateInterval   = 6 * time.Hour
	SyncHeartBeatInterval     = 100 * time.Millisecond
	RequestBlockInterval      = 200 * time.Millisecond
	ResetBlockInterval        = 500 * time.Millisecond
	RequestEpochInterval      = 200 * time.Millisecond
	ResetEpochInterval        = 500 * time.Millisecond
	RequestChainInterval      = 200 * time.Millisecond
	ResetChainInterval        = 500 * time.Millisecond
	RequestNodeInterval       = 200 * time.Millisecond
	ResetNodeInterval         = 500 * time.Millisecond
	RequestAccountInterval    = 200 * time.Millisecond
	ResetAccountStateInterval = 500 * time.Millisecond
)

type ForceSyncRequestType uint32

const (
	ForceSyncBlocks ForceSyncRequestType = iota
	ForceSyncStates
	ForceSyncChains
	ForceSyncNodes
	ForceSyncAccounts
)

type Syncer interface {
	UpdateHandler(handler SyncHandler)

	Marshaler() codec.Marshaler

	ForceSync(ft ForceSyncRequestType, list []interface{}) error

	GetSyncState(nodeID string) (*SyncInfoResponse, error)

	Start(sysActor *actor.ActorSystem, network network.Network) error

	Stop()
}

type syncer struct {
	log                  tplog.Logger
	level                tplogcmm.LogLevel
	handler              *syncHandler
	marshaler            codec.Marshaler
	ctx                  context.Context
	nodeID               string
	query                SyncServant
	wg                   sync.WaitGroup
	SyncHeartBeatTimer   *time.Timer //Forced synchronization timer
	updateNodeScoreTimer *time.Timer
}

func NewSyncer(conf *SyncConfig, level tplogcmm.LogLevel, log tplog.Logger, codecType codec.CodecType,
	ctx context.Context, ledger ledger.Ledger, nodeID string,
	stateQueryService service.StateQueryService,
	blockService service.BlockService) Syncer {
	syncLog := tplog.CreateModuleLogger(level, MOD_NAME, log)
	sy := &syncer{
		log:       syncLog,
		level:     level,
		marshaler: codec.CreateMarshaler(codecType),
		query:     NewSyncServant(stateQueryService, blockService),
		ctx:       ctx,
	}
	sy.handler = NewSyncHandler(conf, sy.log, ctx, sy.marshaler, ledger, nodeID, stateQueryService, blockService)
	return sy
}

func (sa *syncer) UpdateHandler(handler SyncHandler) {

}

func (sa *syncer) Marshaler() codec.Marshaler {
	return sa.marshaler
}

func (sa *syncer) ForceSync(ft ForceSyncRequestType, list []interface{}) error {
	switch ft {
	case ForceSyncBlocks:
		for _, blockHeightI := range list {
			height := blockHeightI.(uint64)
			blockRequest := &BlockRequest{
				NodeID: []byte(sa.nodeID),
				Height: height,
			}
			data, _ := sa.marshaler.Marshal(blockRequest)
			syncMsg := SyncMessage{
				MsgType: SyncMessage_BlockRequest,
				Data:    data,
			}

			syncData, _ := sa.marshaler.Marshal(&syncMsg)
			sa.query.Send(sa.ctx, protocol.SyncProtocolID_Block, MOD_NAME, syncData)
		}
	case ForceSyncStates:
		for _, versionI := range list {
			version := versionI.(uint64)
			stateRequest := &StateRequest{
				NodeID:       []byte(sa.nodeID),
				StateVersion: version,
			}
			data, _ := sa.marshaler.Marshal(stateRequest)

			syncMsg := SyncMessage{
				MsgType: SyncMessage_StateRequest,
				Data:    data,
			}
			syncData, _ := sa.marshaler.Marshal(&syncMsg)
			sa.query.Send(sa.ctx, protocol.AsyncSendProtocolID, MOD_NAME, syncData)
		}
	case ForceSyncChains:
		for _, chainRequestI := range list {
			version := chainRequestI.(uint64)
			chainRequest := &ChainRequest{
				NodeID:       []byte(sa.nodeID),
				StateVersion: version}
			data, _ := sa.marshaler.Marshal(chainRequest)
			syncMsg := SyncMessage{
				MsgType: SyncMessage_ChainRequest,
				Data:    data,
			}
			syncData, _ := sa.marshaler.Marshal(&syncMsg)
			sa.query.Send(sa.ctx, protocol.AsyncSendProtocolID, MOD_NAME, syncData)
		}
	case ForceSyncNodes:
		var nodeIDs []string
		for _, nodeIDI := range list {
			nodeID := nodeIDI.(string)
			nodeIDs = append(nodeIDs, nodeID)
		}
		tmpRoot, err := sa.query.StateRoot()
		if err != nil {
			return err
		}
		tmpVersion, err := sa.query.StateLatestVersion()
		if err != nil {
			return err
		}
		tmpNodeRoot, err := sa.query.GetNodeRoot()
		if err != nil {
			return err
		}
		nodeIDsData, _ := sa.marshaler.Marshal(nodeIDs)
		nodeIDsRequest := &NodeIDsRequest{
			NodeID:       []byte(sa.nodeID),
			StateRoot:    tmpRoot,
			StateVersion: tmpVersion,
			NodeRoot:     tmpNodeRoot,
			NodeIDsData:  nodeIDsData,
		}

		data, _ := sa.marshaler.Marshal(nodeIDsRequest)
		zipData, _ := CompressBytes(data, GZIP)
		syncMsgNode := SyncMessage{
			MsgType: SyncMessage_NodeIDsRequest,
			Data:    zipData,
		}
		syncDataNodeIDsRequest, _ := sa.marshaler.Marshal(&syncMsgNode)
		sa.query.Send(sa.ctx, protocol.AsyncSendProtocolID, MOD_NAME, syncDataNodeIDsRequest)
	case ForceSyncAccounts:
		var accountAddrs []types.Address
		for _, accI := range list {
			acc := accI.(types.Address)
			accountAddrs = append(accountAddrs, acc)
		}
		tmpRoot, err := sa.query.StateRoot()
		if err != nil {
			return err
		}
		tmpVersion, err := sa.query.StateLatestVersion()
		if err != nil {
			return err
		}
		tmpAccountRoot, err := sa.query.GetAccountRoot()
		if err != nil {
			return err
		}
		addrsData, _ := sa.marshaler.Marshal(accountAddrs)
		accountResponse := &AccountAddrsRequest{
			NodeID:           []byte(sa.nodeID),
			StateRoot:        tmpRoot,
			StateVersion:     tmpVersion,
			AccountRoot:      tmpAccountRoot,
			AccountAddrsData: addrsData,
		}
		data, _ := sa.marshaler.Marshal(accountResponse)
		zipData, _ := CompressBytes(data, GZIP)
		syncMsgAccount := SyncMessage{
			MsgType: SyncMessage_AccountResponse,
			Data:    zipData,
		}
		syncDataAccount, _ := sa.marshaler.Marshal(&syncMsgAccount)
		sa.query.Send(sa.ctx, protocol.AsyncSendProtocolID, MOD_NAME, syncDataAccount)

	}
	return nil
}

func (sa *syncer) GetSyncState(nodeID string) (*SyncInfoResponse, error) {

	request := &SyncInfoRequest{
		NodeID:       []byte(sa.nodeID),
		Height:       0,
		StateVersion: 0,
	}
	data, _ := sa.marshaler.Marshal(request)
	msgSyncInfo := SyncMessage{
		MsgType: SyncMessage_SyncInfoReaquest,
		Data:    data,
	}
	syncInfoData, _ := sa.marshaler.Marshal(&msgSyncInfo)

	sendResponse, err := sa.query.SendWithResponse(context.WithValue(sa.ctx, tpnetcmn.NetContextKey_PeerList, []string{nodeID}),
		protocol.AsyncSendProtocolID, MOD_NAME, syncInfoData)
	if err != nil {
		return nil, err
	}
	var syncInfo *SyncInfoResponse
	if err := sendResponse[0].Err; err != nil {
		return nil, err
	}
	err = sa.marshaler.Unmarshal(sendResponse[0].RespData, syncInfo)
	if err != nil {
		return nil, err
	}
	return syncInfo, nil
}

func (sa *syncer) HandleHeartBeatRequest(msg *HeartBeatRequest) error {
	return sa.handler.HandleHeartBeatRequest(msg)
}

func (sa *syncer) handleBlockRequest(msg *BlockRequest) error {
	return sa.handler.HandleBlockRequest(msg)
}

func (sa *syncer) handleBlockResponse(msg *BlockResponse) error {
	return sa.handler.HandleBlockResponse(msg)
}
func (sa *syncer) handleStateRequest(msg *StateRequest) error {
	return sa.handler.HandleStateRequest(msg)
}

func (sa *syncer) handleEpochResponse(msg *EpochResponse) error {
	return sa.handler.HandleEpochResponse(msg)
}

func (sa *syncer) handleNodeIDsRequest(msg *NodeIDsRequest) error {
	return sa.handler.HandleNodeIDsRequest(msg)
}
func (sa *syncer) handleNodeRequest(msg *NodeRequest) error {
	return sa.handler.HandleNodeRequest(msg)
}

func (sa *syncer) handleNodeResponse(msg *NodeResponse) error {
	return sa.handler.HandleNodeResponse(msg)
}

func (sa *syncer) handleChainRequest(msg *ChainRequest) error {
	return sa.handler.HandleChainRequest(msg)
}

func (sa *syncer) handleChainResponse(msg *ChainResponse) error {
	return sa.handler.HandleChainResponse(msg)
}

func (sa *syncer) handleAccountAddrsRequest(msg *AccountAddrsRequest) error {
	return sa.handler.HandleAccountAddrsRequest(msg)
}

func (sa *syncer) handleAccountRequest(msg *AccountRequest) error {
	return sa.handler.HandleAccountRequest(msg)
}

func (sa *syncer) handleAccountResponse(msg *AccountResponse) error {
	return sa.handler.HandleAccountResponse(msg)
}

func (sa *syncer) Start(sysActor *actor.ActorSystem, network network.Network) error {
	defer sa.wg.Wait()
	actorPID, err := CreateSyncActor(sa.level, sa.log, sysActor, sa)
	if err != nil {
		sa.log.Panicf("CreateSyncActor error: %v", err)
		return err
	}
	network.RegisterModule(MOD_NAME, actorPID, sa.marshaler)

	sa.wg.Add(1)
	go sa.loop()

	sa.wg.Add(1)
	go sa.loopFetchBlocks()
	sa.wg.Add(1)
	go sa.loopDoSyncBlocks()
	sa.wg.Add(1)
	go sa.loopFetchEpochs()
	sa.wg.Add(1)
	go sa.loopDoSyncEpochs()
	sa.wg.Add(1)
	go sa.loopFetchNodes()
	sa.wg.Add(1)
	go sa.loopDoSyncNodes()
	sa.wg.Add(1)
	go sa.loopFetchChain()
	sa.wg.Add(1)
	go sa.loopDoSyncChain()
	sa.wg.Add(1)
	go sa.loopFetchAccounts()
	sa.wg.Add(1)
	go sa.loopDoSyncAccounts()

	return nil
}

func (sa *syncer) loop() {
	sa.SyncHeartBeatTimer = time.NewTimer(SyncHeartBeatInterval)
	sa.updateNodeScoreTimer = time.NewTimer(NodeScoreUpdateInterval)
	defer sa.SyncHeartBeatTimer.Stop()
	for {

		bytesSyncHeartBeat := sa.getCurrentHeartBeatByte()

		select {
		case <-sa.updateNodeScoreTimer.C:
			sa.query.ReNewNodeScores()
		case <-sa.SyncHeartBeatTimer.C:
			sa.query.Send(sa.ctx, protocol.HeartBeatPtotocolID, MOD_NAME, bytesSyncHeartBeat)
			sa.SyncHeartBeatTimer.Reset(SyncHeartBeatInterval)
		case <-sa.handler.quitSync:
			//quit synchronization requires stopping the insertion of data into the blockchain
			//and then stopping the downloader.
			sa.handler.blockDownloader.chanQuitSync <- struct{}{}
			sa.handler.epochDownloader.chanQuitSync <- struct{}{}
			sa.handler.nodeDownloader.chanQuitSync <- struct{}{}
			sa.handler.chainDownloader.chanQuitSync <- struct{}{}
			sa.handler.accountDownloader.chanQuitSync <- struct{}{}
			return
		}
	}

}
func (sa *syncer) loopFetchBlocks() {
	defer sa.wg.Done()
	sa.handler.blockDownloader.loopFetchBlocks()
}
func (sa *syncer) loopDoSyncBlocks() {
	defer sa.wg.Done()
	sa.handler.blockDownloader.loopDoSyncBlocks()
}
func (sa *syncer) loopFetchEpochs() {
	defer sa.wg.Done()
	sa.handler.epochDownloader.loopFetchEpochs()
}
func (sa *syncer) loopDoSyncEpochs() {
	defer sa.wg.Done()
	sa.handler.epochDownloader.loopDoSyncEpochs()
}
func (sa *syncer) loopFetchNodes() {
	defer sa.wg.Done()
	sa.handler.nodeDownloader.loopFetchNodes()
}

func (sa *syncer) loopDoSyncNodes() {
	defer sa.wg.Done()
	sa.handler.nodeDownloader.loopDoSyncNodes()
}
func (sa *syncer) loopFetchChain() {
	defer sa.wg.Done()
	sa.handler.chainDownloader.loopFetchChain()
}
func (sa *syncer) loopDoSyncChain() {
	defer sa.wg.Done()
	sa.handler.chainDownloader.loopDoSyncChain()
}
func (sa *syncer) loopFetchAccounts() {
	defer sa.wg.Done()
	sa.handler.accountDownloader.loopFetchAccounts()
}
func (sa *syncer) loopDoSyncAccounts() {
	defer sa.wg.Done()
	sa.handler.accountDownloader.loopDoSyncAccounts()
}
func (sa *syncer) getCurrentHeartBeatByte() []byte {
	curHeight, err := sa.query.GetLatestHeight()
	if err != nil {
		sa.log.Errorf("query current Height Error:", err)
	}
	curStateRoot, err := sa.query.StateRoot()
	if err != nil {
		sa.log.Errorf("query current account state root Error:", err)
	}
	curStateVersion, err := sa.query.StateLatestVersion()
	if err != nil {
		sa.log.Errorf("query current chain state root Error:", err)
	}

	heartBeatRequest := &HeartBeatRequest{
		BlockHeight: curHeight,
		StateRoot:   curStateRoot,
		SateVersion: curStateVersion,
	}

	data, _ := sa.marshaler.Marshal(heartBeatRequest)
	syncMsg := SyncMessage{
		MsgType: SyncMessage_HeartBeatRequest,
		Data:    data,
	}
	syncData, _ := sa.marshaler.Marshal(&syncMsg)
	return syncData
}

func (sa *syncer) dispatch(context actor.Context, data []byte) {
	var syncMsg SyncMessage
	err := sa.marshaler.Unmarshal(data, &syncMsg)
	if err != nil {
		sa.log.Errorf("Syncer receive invalid data %v", data)
		return
	}
	switch syncMsg.MsgType {
	case SyncMessage_SyncInfoReaquest:
		var msg SyncInfoRequest
		err = sa.marshaler.Unmarshal(syncMsg.Data, &msg)
		if err != nil {
			sa.log.Errorf("Syncer unmarshal msg %d err %v", syncMsg.MsgType, err)
			return
		}
		curHeight, err := sa.query.GetLatestHeight()
		if err != nil {
			sa.log.Errorf("Syncer GetLatestHeight err %v", err)
			return
		}
		curVersion, err := sa.query.StateLatestVersion()
		if err != nil {
			sa.log.Errorf("Syncer StateLatestVersion err %v", err)
			return
		}
		stateRoot, err := sa.query.StateRoot()
		if err != nil {
			sa.log.Errorf("Syncer StateRoot err %v", err)
			return
		}
		accountRoot, err := sa.query.GetAccountRoot()
		if err != nil {
			sa.log.Errorf("Syncer GetAccountRoot err %v", err)
			return
		}
		chainRoot, err := sa.query.GetChainRoot()
		if err != nil {
			sa.log.Errorf("Syncer GetChainRoot err %v", err)
			return
		}
		nodeRoot, err := sa.query.GetNodeRoot()
		if err != nil {
			sa.log.Errorf("Syncer GetNodeRoot err %v", err)
			return
		}
		epochRoot, err := sa.query.GetEpochRoot()
		if err != nil {
			sa.log.Errorf("Syncer GetEpochRoot err %v", err)
			return
		}
		response := &SyncInfoResponse{
			NodeID:       []byte(sa.nodeID),
			Height:       curHeight,
			StateVersion: curVersion,
			StateRoot:    stateRoot,
			AccountRoot:  accountRoot,
			ChainRoot:    chainRoot,
			NodeRoot:     nodeRoot,
			EpochRoot:    epochRoot,
		}

		var respData tpnetmsg.SendResponse
		if err != nil {
			respData.Err = err
		}
		respData.NodeID = sa.nodeID
		respData.RespData, _ = sa.marshaler.Marshal(response)
		context.Respond(&respData)

	case SyncMessage_HeartBeatRequest:
		var msg HeartBeatRequest
		err = sa.marshaler.Unmarshal(syncMsg.Data, &msg)
		if err != nil {
			sa.log.Errorf("Syncer unmarshal msg %d err %v", syncMsg.MsgType, err)
			return
		}
		sa.HandleHeartBeatRequest(&msg)

	case SyncMessage_BlockRequest:
		var msg BlockRequest
		err = sa.marshaler.Unmarshal(syncMsg.Data, &msg)
		if err != nil {
			sa.log.Errorf("Syncer unmarshal msg %d err %v", syncMsg.MsgType, err)
			return
		}
		sa.handleBlockRequest(&msg)
	case SyncMessage_BlockResponse:
		unZipData, err := UnCompressBytes(syncMsg.Data, GZIP)
		if err != nil {
			sa.log.Errorf("Syncer UnCompressBytes msg %d err %v", syncMsg.MsgType, err)
			return
		}
		var msg BlockResponse
		err = sa.marshaler.Unmarshal(unZipData, &msg)
		if err != nil {
			sa.log.Errorf("Syncer unmarshal msg %d err %v", syncMsg.MsgType, err)
			return
		}
		sa.handleBlockResponse(&msg)
	case SyncMessage_StateRequest:
		unZipData, err := UnCompressBytes(syncMsg.Data, GZIP)
		if err != nil {
			sa.log.Errorf("Syncer UnCompressBytes msg %d err %v", syncMsg.MsgType, err)
			return
		}
		var msg StateRequest
		err = sa.marshaler.Unmarshal(unZipData, &msg)
		if err != nil {
			sa.log.Errorf("Syncer unmarshal msg %d err %v", syncMsg.MsgType, err)
			return
		}
		sa.handleStateRequest(&msg)
	case SyncMessage_EpochResponse:
		var msg EpochResponse
		err = sa.marshaler.Unmarshal(syncMsg.Data, &msg)
		if err != nil {
			sa.log.Errorf("Syncer unmarshal msg %d err %v", syncMsg.MsgType, err)
			return
		}
		sa.handleEpochResponse(&msg)
	case SyncMessage_NodeIDsRequest:
		unZipData, err := UnCompressBytes(syncMsg.Data, GZIP)
		if err != nil {
			sa.log.Errorf("Syncer UnCompressBytes msg %d err %v", syncMsg.MsgType, err)
			return
		}
		var msg NodeIDsRequest
		err = sa.marshaler.Unmarshal(unZipData, &msg)
		if err != nil {
			sa.log.Errorf("Syncer unmarshal msg %d err %v", syncMsg.MsgType, err)
			return
		}
		sa.handleNodeIDsRequest(&msg)
	case SyncMessage_NodeRequest:
		unZipData, err := UnCompressBytes(syncMsg.Data, GZIP)
		if err != nil {
			sa.log.Errorf("Syncer UnCompressBytes msg %d err %v", syncMsg.MsgType, err)
			return
		}
		var msg NodeRequest
		err = sa.marshaler.Unmarshal(unZipData, &msg)
		if err != nil {
			sa.log.Errorf("Syncer unmarshal msg %d err %v", syncMsg.MsgType, err)
			return
		}
		sa.handleNodeRequest(&msg)
	case SyncMessage_NodeResponse:
		unZipData, err := UnCompressBytes(syncMsg.Data, GZIP)
		if err != nil {
			sa.log.Errorf("Syncer UnCompressBytes msg %d err %v", syncMsg.MsgType, err)
			return
		}
		var msg NodeResponse
		err = sa.marshaler.Unmarshal(unZipData, &msg)
		if err != nil {
			sa.log.Errorf("Syncer unmarshal msg %d err %v", syncMsg.MsgType, err)
			return
		}
		sa.handleNodeResponse(&msg)
	case SyncMessage_ChainRequest:
		var msg ChainRequest
		err = sa.marshaler.Unmarshal(syncMsg.Data, &msg)
		if err != nil {
			sa.log.Errorf("Syncer unmarshal msg %d err %v", syncMsg.MsgType, err)
			return
		}
		sa.handleChainRequest(&msg)
	case SyncMessage_ChainResponse:
		unZipData, err := UnCompressBytes(syncMsg.Data, GZIP)
		if err != nil {
			sa.log.Errorf("Syncer UnCompressBytes msg %d err %v", syncMsg.MsgType, err)
			return
		}
		var msg ChainResponse
		err = sa.marshaler.Unmarshal(unZipData, &msg)
		if err != nil {
			sa.log.Errorf("Syncer unmarshal msg %d err %v", syncMsg.MsgType, err)
			return
		}
		sa.handleChainResponse(&msg)
	case SyncMessage_AccountAddrsRequest:
		unZipData, err := UnCompressBytes(syncMsg.Data, GZIP)
		if err != nil {
			sa.log.Errorf("Syncer UnCompressBytes msg %d err %v", syncMsg.MsgType, err)
			return
		}
		var msg AccountAddrsRequest
		err = sa.marshaler.Unmarshal(unZipData, &msg)
		if err != nil {
			sa.log.Errorf("Syncer unmarshal msg %d err %v", syncMsg.MsgType, err)
			return
		}
		sa.handleAccountAddrsRequest(&msg)
	case SyncMessage_AccountRequest:
		unZipData, err := UnCompressBytes(syncMsg.Data, GZIP)
		if err != nil {
			sa.log.Errorf("Syncer UnCompressBytes msg %d err %v", syncMsg.MsgType, err)
			return
		}
		var msg AccountRequest
		err = sa.marshaler.Unmarshal(unZipData, &msg)
		if err != nil {
			sa.log.Errorf("Syncer unmarshal msg %d err %v", syncMsg.MsgType, err)
			return
		}
		sa.handleAccountRequest(&msg)
	case SyncMessage_AccountResponse:
		unZipData, err := UnCompressBytes(syncMsg.Data, GZIP)
		if err != nil {
			sa.log.Errorf("Syncer UnCompressBytes msg %d err %v", syncMsg.MsgType, err)
			return
		}
		var msg AccountResponse
		err = sa.marshaler.Unmarshal(unZipData, &msg)
		if err != nil {
			sa.log.Errorf("Syncer unmarshal msg %d err %v", syncMsg.MsgType, err)
			return
		}
		sa.handleAccountResponse(&msg)
	default:
		sa.log.Errorf("Syncer receive invalid msg %d", syncMsg.MsgType)
		return
	}
}

func (sync *syncer) Stop() {
	panic("implement me")
}
