package sync

import (
	"bytes"
	"context"
	"errors"
	"reflect"
	"sync/atomic"
	"time"

	"github.com/TopiaNetwork/topia/account"
	tpchaintypes "github.com/TopiaNetwork/topia/chain/types"
	"github.com/TopiaNetwork/topia/codec"
	tpcmm "github.com/TopiaNetwork/topia/common"
	"github.com/TopiaNetwork/topia/crypt/types"
	"github.com/TopiaNetwork/topia/ledger"
	tplog "github.com/TopiaNetwork/topia/log"
	"github.com/TopiaNetwork/topia/network/message"
	"github.com/TopiaNetwork/topia/network/protocol"
	"github.com/TopiaNetwork/topia/state"
	"github.com/TopiaNetwork/topia/transaction"
	txbasic "github.com/TopiaNetwork/topia/transaction/basic"
)

var (
	ErrParentBlockHash = errors.New("error parent blockHash")
	ErrHeight          = errors.New("error block height")
	ErrDiffRoot        = errors.New("error for different root")
)

const (
	FetchBlockChanSize   = MaxSyncHeights
	FetchEpochChanSize   = MaxSyncHeights
	FetchNodeChanSize    = 1024 * MaxSyncHeights
	FetchChainChanSize   = MaxSyncHeights
	FetchAccountChanSize = 1024 * MaxSyncHeights
)

type Downloader struct {
	mode               SyncMode
	resetCurLevel      bool
	forceRequest       bool
	log                tplog.Logger
	marshaler          codec.Marshaler
	ledger             ledger.Ledger
	servant            SyncServant
	ctx                context.Context
	chanQuitSync       chan struct{}
	DoneSync           chan error
	requestInterval    *time.Timer
	resetStateInterval *time.Timer
	reNewScoreInterval *time.Timer
}

func NewDownloader(conf *SyncConfig, log tplog.Logger, marshaler codec.Marshaler, ctx context.Context, ledger ledger.Ledger) *Downloader {
	downloader := &Downloader{
		mode:         conf.Mode,
		log:          log,
		marshaler:    marshaler,
		ctx:          ctx,
		ledger:       ledger,
		chanQuitSync: make(chan struct{}),
		DoneSync:     make(chan error, 1),
	}
	return downloader
}

type BlockDownloader struct {
	*Downloader
	nodeID    string
	txServant txbasic.TransactionServant
	//savedBlockHeights *IdBitmap //save block height for block saved in queue
	chanFetchBlock chan *FreeItem
	blockSyncQueue *LockFreePriorityQueue //look free height sorted queue for blocks

}

func NewBlockDownloader(conf *SyncConfig, log tplog.Logger, marshaler codec.Marshaler, ctx context.Context, ledger ledger.Ledger, nodeId string) *BlockDownloader {
	downloader := &BlockDownloader{
		Downloader:     NewDownloader(conf, log, marshaler, ctx, ledger),
		nodeID:         nodeId,
		blockSyncQueue: NewLKQueue(),
		chanFetchBlock: make(chan *FreeItem, FetchBlockChanSize),
	}
	return downloader
}

func (bd *BlockDownloader) loopFetchBlocks() {
	for {
		select {
		case v := <-bd.chanFetchBlock:

			bd.blockSyncQueue.Push(v, v.value.(*tpchaintypes.Block).Head.Height)

		case <-bd.chanQuitSync:
			return
		default:
			continue
		}
	}
}

func (bd *BlockDownloader) loopDoSyncBlocks() {
	bd.requestInterval = time.NewTimer(RequestBlockInterval)
	bd.resetStateInterval = time.NewTimer(ResetBlockInterval)
	defer bd.requestInterval.Stop()
	defer bd.resetStateInterval.Stop()
	for {

	getAgain:
		height, err := bd.blockSyncQueue.GetHeadPriority()
		if err != nil || height == 0 {
			goto getAgain
		}
		curBlockHeight, err := bd.servant.GetLatestHeight()
		if err != nil {
			bd.log.Errorf("get latest block height error:", err)
		}
		if bd.forceRequest {
			bd.forceRequestBlock(curBlockHeight, height)
			bd.forceRequest = false
		}

		if height == curBlockHeight+1 {
			item := bd.blockSyncQueue.PopHead()
			block := item.value.(*tpchaintypes.Block)
			if block.Head.Height > height {
				bd.blockSyncQueue.Push(item, block.Head.Height)
			} else if block.Head.Height < height {
				goto getAgain
			}

			//do block sync
			var tx *txbasic.Transaction
			var preTx *txbasic.Transaction
			switch bd.mode {
			case FastSync:
				if _, err := bd.blockVerify(block); err != nil {
					bd.resetCurLevel = true
					bd.forceRequest = true
					goto getAgain
				}

				for i, v := range block.Data.Txs {
					err := bd.marshaler.Unmarshal(v, &tx)
					if err != nil {
						bd.resetCurLevel = true
						bd.forceRequest = true
						goto getAgain
					}
					addr := types.Address(tx.Head.FromAddr)
					if i == 0 {
						curNonce, err := bd.servant.GetNonce(addr)
						if err != nil {
							if tx.Head.Nonce != 1 {
								bd.resetCurLevel = true
								bd.forceRequest = true
								goto getAgain
							}
						}
						if tx.Head.Nonce != curNonce+1 {
							bd.resetCurLevel = true
							bd.forceRequest = true
							goto getAgain
						}
					} else if i > 0 {
						preTxByte := block.Data.Txs[i-1]
						err := bd.marshaler.Unmarshal(preTxByte, &preTx)
						if err != nil {
							bd.resetCurLevel = true
							bd.forceRequest = true
							goto getAgain
						}
						if bytes.Equal(tx.Head.FromAddr, preTx.Head.FromAddr) {
							if tx.Head.Nonce != preTx.Head.Nonce+1 {
								bd.resetCurLevel = true
								bd.forceRequest = true
								goto getAgain
							}
						} else {
							curNonce, err := bd.servant.GetNonce(addr)
							if err != nil {
								if tx.Head.Nonce != 1 {
									bd.resetCurLevel = true
									bd.forceRequest = true
									goto getAgain
								}
							}
							if tx.Head.Nonce != curNonce+1 {
								bd.resetCurLevel = true
								bd.forceRequest = true
								goto getAgain
							}
						}
					}
				}
				blockStore := bd.servant.GetBlockStore()
				err := blockStore.CommitBlock(block)
				if err != nil {
					goto getAgain
				}
			case FullSync:
				var txs []*txbasic.Transaction
				var tx *txbasic.Transaction
				var txsResult []txbasic.TransactionResult
				for i, v := range block.Data.Txs {

					err := bd.marshaler.Unmarshal(v, &tx)
					if err != nil {
						bd.resetCurLevel = true
						bd.forceRequest = true
						goto getAgain
					}
					addr := types.Address(tx.Head.FromAddr)
					if i == 0 {
						curNonce, err := bd.servant.GetNonce(addr)
						if err != nil {
							if tx.Head.Nonce != 1 {
								bd.resetCurLevel = true
								bd.forceRequest = true
								goto getAgain
							}
						}
						if tx.Head.Nonce != curNonce+1 {
							bd.resetCurLevel = true
							bd.forceRequest = true
							goto getAgain
						}
					} else if i > 0 {
						preTxByte := block.Data.Txs[i-1]
						err := bd.marshaler.Unmarshal(preTxByte, &preTx)
						if err != nil {
							bd.resetCurLevel = true
							bd.forceRequest = true
							goto getAgain
						}
						if bytes.Equal(tx.Head.FromAddr, preTx.Head.FromAddr) {
							if tx.Head.Nonce != preTx.Head.Nonce+1 {
								bd.resetCurLevel = true
								bd.forceRequest = true
								goto getAgain
							}
						} else {
							curNonce, err := bd.servant.GetNonce(addr)
							if err != nil {
								if tx.Head.Nonce != 1 {
									bd.resetCurLevel = true
									bd.forceRequest = true
									goto getAgain
								}
							}
							if tx.Head.Nonce != curNonce+1 {
								bd.resetCurLevel = true
								bd.forceRequest = true
								goto getAgain
							}
						}
					}

					action := transaction.CreatTransactionAction(tx)
					result := bd.txVerify(tx)
					invalidCnt := 0
					switch result {
					case message.ValidationAccept:
						invalidCnt = 0
					case message.ValidationIgnore:
						invalidCnt = 1
					case message.ValidationReject:
						invalidCnt = 1
					default:
						invalidCnt = 1
					}
					if invalidCnt != 0 {
						bd.forceRequest = true
						bd.resetCurLevel = true
						goto getAgain
					}
					txRt := action.Execute(bd.ctx, bd.log, bd.nodeID, bd.txServant)
					txsResult = append(txsResult, *txRt)
				}

				TxRSRoot := txbasic.TxResultRoot(txsResult, txs)
				latestBlockResult, err := bd.servant.GetLatestBlockResult()
				if err != nil {
					bd.forceRequest = true
					bd.resetCurLevel = true
					bd.log.Errorf("Can't get latest block result : %v", err)
					goto getAgain
				}
				blockRSHash, err := latestBlockResult.HashBytes(tpcmm.NewBlake2bHasher(0), bd.marshaler)
				if err != nil {
					bd.forceRequest = true
					bd.resetCurLevel = true
					bd.log.Errorf("Can't get latest block result hash: %v", err)
				}
				blockResultHead := &tpchaintypes.BlockResultHead{
					Version:          block.Head.Version,
					PrevBlockResult:  blockRSHash,
					BlockHash:        block.Head.Hash,
					TxResultHashRoot: tpcmm.BytesCopy(TxRSRoot),
					Status:           tpchaintypes.BlockResultHead_OK,
				}
				blockResultData := &tpchaintypes.BlockResultData{
					Version: block.Head.Version,
				}
				for r := 0; r < len(txsResult); r++ {
					txRSBytes, _ := txsResult[r].HashBytes()
					blockResultData.TxResults = append(blockResultData.TxResults, txRSBytes)
				}
				blockResult := &tpchaintypes.BlockResult{
					Head: blockResultHead,
					Data: blockResultData,
				}
			getLatestBlock:
				curBlock, err := bd.servant.GetLatestBlock()
				if err != nil {
					bd.forceRequest = true
					bd.resetCurLevel = true
					goto getAgain
				}
				curHeight := curBlock.Head.Height
				if curHeight == block.Head.Height {
					curBlockResult, err := bd.servant.GetLatestBlockResult()
					if err != nil {
						bd.forceRequest = true
						bd.resetCurLevel = true
						goto getAgain
					}
					if !reflect.DeepEqual(blockResult, curBlockResult) {
						bd.forceRequest = true
						bd.resetCurLevel = true
						goto getAgain
					}
					blockState := bd.servant.GetBlockStore()
					err = blockState.CommitBlock(block)
					if err != nil {
						goto getAgain
					}

				} else if curHeight < block.Head.Height {
					for h := block.Head.Height; h <= curHeight; h++ {
						stateRequest := &StateRequest{
							NodeID:       []byte(bd.nodeID),
							StateVersion: h,
						}
						data, _ := bd.marshaler.Marshal(stateRequest)
						syncMsg := SyncMessage{
							MsgType: SyncMessage_StateRequest,
							Data:    data,
						}
						syncData, _ := bd.marshaler.Marshal(syncMsg)
						bd.servant.Send(bd.ctx, protocol.AsyncSendProtocolID, MOD_NAME, syncData)
						bd.requestInterval.Reset(RequestNodeInterval)
					}
					goto getLatestBlock
				} else if curHeight > block.Head.Height {
					tmpState := state.CreateCompositionStateReadonlyAt(bd.log, bd.ledger, block.Head.Height)
					tmpBlockResult, err := tmpState.GetLatestBlockResult()
					if err != nil {
						bd.log.Errorf("download doSyncBlock error GetLatestBlockResult", err)
						goto getAgain
					}
					if !reflect.DeepEqual(tmpBlockResult, blockResult) {
						goto getAgain
					}
					blockStore := bd.servant.GetBlockStore()
					err = blockStore.CommitBlock(block)
					if err != nil {
						goto getAgain
					}
				}
			}
			bd.requestInterval.Reset(RequestBlockInterval)
			bd.resetStateInterval.Reset(ResetBlockInterval)
		} else if curBlockHeight >= height {
			bd.blockSyncQueue.PopHead()
			goto getAgain
		}

		select {
		case <-bd.chanQuitSync:
			return
		case <-bd.requestInterval.C:
			bd.forceRequestBlock(curBlockHeight, height)
		case <-bd.resetStateInterval.C:
			bd.resetCurLevel = true
			bd.forceRequestBlock(curBlockHeight, height)
			bd.resetStateInterval.Reset(ResetBlockInterval)
		default:
			continue
		}
	}
}
func (bd *BlockDownloader) blockVerify(block *tpchaintypes.Block) (bool, error) {
	curBlock, err := bd.servant.GetLatestBlock()
	if err != nil {
		return false, err
	}
	if !bytes.Equal(block.Head.ParentBlockHash, curBlock.Head.Hash) {
		return false, ErrParentBlockHash
	}
	if block.Head.Height != curBlock.Head.Height+1 {
		return false, ErrHeight
	}
	return true, nil
}

func (bd *BlockDownloader) txVerify(tx *txbasic.Transaction) message.ValidationResult {
	ac := transaction.CreatTransactionAction(tx)
	verifyResult := ac.Verify(bd.ctx, bd.log, bd.nodeID, nil)
	switch verifyResult {
	case txbasic.VerifyResult_Accept:
		return message.ValidationAccept
	case txbasic.VerifyResult_Ignore:
		return message.ValidationIgnore
	case txbasic.VerifyResult_Reject:
		return message.ValidationReject
	default:
		return message.ValidationReject
	}
}

func (bd *BlockDownloader) forceRequestBlock(curBlockHeight uint64, height uint64) {

	blockRequest := &BlockRequest{
		NodeID: []byte(bd.nodeID),
		Height: curBlockHeight,
	}
	data, _ := bd.marshaler.Marshal(blockRequest)
	syncMsg := SyncMessage{
		MsgType: SyncMessage_BlockRequest,
		Data:    data,
	}

	syncData, _ := bd.marshaler.Marshal(&syncMsg)
	bd.servant.Send(bd.ctx, protocol.SyncProtocolID_Block, MOD_NAME, syncData)
	bd.requestInterval.Reset(RequestBlockInterval)
}

type EpochDownloader struct {
	*Downloader
	nodeID          string
	curSyncVersion  uint64
	remoteEpochRoot []byte
	//savedEpochs        *IdBitmap //save epochs for epochInfos saved in queue
	chanFetchEpochInfo chan *FreeItem
	epochInfoSyncQueue *LockFreePriorityQueue //look free height sorted queue for EpochInfos
}

func NewEpochDownloader(conf *SyncConfig, log tplog.Logger, marshaler codec.Marshaler, ctx context.Context, ledger ledger.Ledger, nodeID string) *EpochDownloader {
	downloader := &EpochDownloader{
		Downloader: NewDownloader(conf, log, marshaler, ctx, ledger),
		nodeID:     nodeID,
		//savedEpochs:        new(IdBitmap),
		chanFetchEpochInfo: make(chan *FreeItem, FetchEpochChanSize),
		epochInfoSyncQueue: NewLKQueue(),
	}
	return downloader
}

func (ed *EpochDownloader) loopFetchEpochs() {
	for {
		select {
		case v := <-ed.chanFetchEpochInfo:
			ed.epochInfoSyncQueue.Push(v, v.value.(*EpochResponse).StateVersion)
		case <-ed.chanQuitSync:
			return
		default:
			continue
		}
	}
}

func (ed *EpochDownloader) loopDoSyncEpochs() {
	ed.requestInterval = time.NewTimer(RequestEpochInterval)
	ed.resetStateInterval = time.NewTimer(ResetEpochInterval)
	defer ed.requestInterval.Stop()
	defer ed.resetStateInterval.Stop()
	curStateVersion, err := ed.servant.StateLatestVersion()
	if err != nil {
		ed.log.Errorf("epoch downLoader get StateLatestVersion error:", err)
	}
	ed.curSyncVersion = curStateVersion

	for {
	getAgain:
		version, err := ed.epochInfoSyncQueue.GetHeadPriority()
		if err != nil || version == 0 {
			if ed.DoneSync != nil {
				<-ed.DoneSync
			}
			goto getAgain
		}

		curEpochInfo, err := ed.servant.GetLatestEpoch()
		if err != nil {
			goto getAgain
		}
		curEpoch := curEpochInfo.Epoch
		curStateVersion, err = ed.servant.StateLatestVersion()
		if err != nil {
			goto getAgain
		}
		if ed.forceRequest {
			if err = ed.forceRequestCurEpoch(); err != nil {
				ed.resetCurLevel = true
				goto getAgain
			}
			ed.forceRequest = false
		}
		if version == curStateVersion {
			item := ed.epochInfoSyncQueue.PopHead()
			epochResponse := item.value.(*EpochResponse)
			if epochResponse.Epoch == curEpoch {
				atomic.AddUint64(&ed.curSyncVersion, 1)
			} else if epochResponse.Epoch < curEpoch {
				goto getAgain
			}
			epochInfo := &tpcmm.EpochInfo{
				Epoch:          epochResponse.Epoch,
				StartTimeStamp: epochResponse.StartTimeStamp,
				StartHeight:    epochResponse.StartHeight,
			}
			//**write epochInfo to local**
			if epochInfo.Epoch != curEpoch+1 {
				goto getAgain
			}
			err := ed.servant.SetLatestEpoch(epochInfo)
			if err != nil {
				ed.forceRequest = true
				ed.resetCurLevel = true
				goto getAgain
			}

			if bytes.Equal(epochResponse.EpochRoot, ed.remoteEpochRoot) {
				ed.requestInterval.Reset(RequestEpochInterval)
				ed.resetStateInterval.Reset(ResetEpochInterval)
				atomic.AddUint64(&ed.curSyncVersion, 1)
				goto getAgain
			}
		} else if version == curStateVersion+1 && ed.curSyncVersion == curStateVersion+1 {
			item := ed.epochInfoSyncQueue.PopHead()
			epochResponse := item.value.(*EpochResponse)
			if epochResponse.Epoch == curEpoch {
				atomic.AddUint64(&ed.curSyncVersion, 1)
			} else if epochResponse.Epoch < curEpoch {
				goto getAgain
			}
			epochInfo := &tpcmm.EpochInfo{
				Epoch:          epochResponse.Epoch,
				StartTimeStamp: epochResponse.StartTimeStamp,
				StartHeight:    epochResponse.StartHeight,
			}
			//**write epochInfo to local**
			if epochInfo.Epoch != curEpoch+1 {
				goto getAgain
			}
			err := ed.servant.SetLatestEpoch(epochInfo)
			if err != nil {
				ed.forceRequest = true
				ed.resetCurLevel = true
				goto getAgain
			}

			if bytes.Equal(epochResponse.EpochRoot, ed.remoteEpochRoot) {
				ed.requestInterval.Reset(RequestEpochInterval)
				ed.resetStateInterval.Reset(ResetEpochInterval)
				atomic.AddUint64(&ed.curSyncVersion, 1)
				goto getAgain
			}
		}
		select {
		case <-ed.chanQuitSync:
			return
		case <-ed.requestInterval.C:
			err := ed.forceRequestCurEpoch()
			if err != nil {
				ed.resetCurLevel = true
				goto getAgain
			}
		case <-ed.resetStateInterval.C:
			ed.resetCurLevel = true
			err := ed.forceRequestCurEpoch()
			if err != nil {
				ed.resetCurLevel = true
				goto getAgain
			}
			ed.resetStateInterval.Reset(ResetEpochInterval)
		default:
			continue
		}
	}
}
func (ed *EpochDownloader) forceRequestCurEpoch() error {
	if ed.DoneSync == nil {
		ed.requestInterval.Reset(RequestEpochInterval)
	}
	curEpochRoot, err := ed.servant.GetEpochRoot()
	if err != nil {
		ed.log.Errorf("get EpochRoot error: ", err)
		return err
	}
	if bytes.Equal(ed.remoteEpochRoot, curEpochRoot) {
		ed.requestInterval.Reset(RequestEpochInterval)
	}
	curStateRoot, err := ed.servant.StateRoot()
	if err != nil {
		ed.log.Errorf("get StateRoot error: ", err)
		return err
	}
	curStateVersion, err := ed.servant.StateLatestVersion()
	if err != nil {
		ed.log.Errorf("get StateLatestVersion error: ", err)
		return err
	}
	curEpochInfo, err := ed.servant.GetLatestEpoch()
	if err != nil {
		ed.log.Errorf("get LatestEpoch error: ", err)
		return err
	}
	epochRequest := &EpochResponse{
		NodeID:         []byte(ed.nodeID),
		StateRoot:      curStateRoot,
		StateVersion:   curStateVersion,
		EpochRoot:      curEpochRoot,
		Epoch:          curEpochInfo.Epoch,
		StartTimeStamp: curEpochInfo.StartTimeStamp,
		StartHeight:    curEpochInfo.StartHeight,
	}
	data, _ := ed.marshaler.Marshal(epochRequest)
	syncMsg := SyncMessage{
		MsgType: SyncMessage_EpochResponse,
		Data:    data,
	}

	syncData, _ := ed.marshaler.Marshal(&syncMsg)
	ed.servant.Send(ed.ctx, protocol.AsyncSendProtocolID, MOD_NAME, syncData)
	ed.requestInterval.Reset(RequestEpochInterval)
	return nil
}

type NodeDownloader struct {
	*Downloader
	curSyncVersion        uint64
	remoteNodeRoot        []byte
	chanFetchNodeFragment chan *NodeResponse
	nodeFragmentQueue     *LockFreeSetQueue
}

func NewNodeDownloader(conf *SyncConfig, log tplog.Logger, marshaler codec.Marshaler, ctx context.Context, ledger ledger.Ledger) *NodeDownloader {
	downloader := &NodeDownloader{
		Downloader:            NewDownloader(conf, log, marshaler, ctx, ledger),
		chanFetchNodeFragment: make(chan *NodeResponse, FetchNodeChanSize),
		nodeFragmentQueue:     NewLKQueueSet(),
	}
	return downloader
}

func (nd *NodeDownloader) loopFetchNodes() {
	for {
		select {
		case v := <-nd.chanFetchNodeFragment:
			tmpItem := &LKQueueSet{
				Map:       make(map[interface{}]struct{}, 0),
				Priority:  v.StateVersion,
				ListBytes: v.NodesData,
			}
			var list []*tpcmm.NodeInfo
			err := nd.marshaler.Unmarshal(tmpItem.ListBytes, list)
			if err != nil {
				nd.log.Errorf("Unmarshal list bytes err", err)
			}
			var idNode *KeyValueItem
			for _, node := range list {
				idNode.K = node.NodeID
				idNode.V = node
				tmpItem.List = append(tmpItem.List, idNode)
			}
			nd.nodeFragmentQueue.Push(tmpItem, v.StateVersion)
		case <-nd.chanQuitSync:
			return
		default:
			continue
		}
	}
}

func (nd *NodeDownloader) loopDoSyncNodes() {
	nd.requestInterval = time.NewTimer(RequestNodeInterval)
	nd.resetStateInterval = time.NewTimer(ResetNodeInterval)
	defer nd.requestInterval.Stop()
	defer nd.resetStateInterval.Stop()
	curStateVersion, err := nd.servant.StateLatestVersion()
	if err != nil {
		nd.log.Errorf("node downLoader get StateLatestVersion error:", err)
	}
	nd.curSyncVersion = curStateVersion

	for {
	getAgain:
		version, err := nd.nodeFragmentQueue.GetHeadPriority()
		if err != nil || version == 0 {
			if nd.DoneSync != nil {
				<-nd.DoneSync
			}
			goto getAgain
		}
		curStateVersion, err = nd.servant.StateLatestVersion()
		if err != nil || version == 0 {
			goto getAgain
		}

		if nd.forceRequest {
			if err := nd.forceRequestNode(); err != nil {
				nd.resetCurLevel = true
				goto getAgain
			}
			nd.forceRequest = false
		}
		if version == curStateVersion && curStateVersion == nd.curSyncVersion {
			for {
				version, err := nd.nodeFragmentQueue.GetHeadPriority()
				if err != nil || version == 0 {
					goto getAgain
				}
				nodeRoot := nd.nodeFragmentQueue.GetHeadRoot()
				idNodes := nd.nodeFragmentQueue.PopHeadList()
				if idNodes == nil || len(idNodes) == 0 {
					curNodeRoot, err := nd.servant.GetNodeRoot()
					if err != nil {
						goto getAgain
					}
					if bytes.Equal(curNodeRoot, nd.remoteNodeRoot) {
						nd.nodeFragmentQueue.dropHead()
						atomic.AddUint64(&nd.curSyncVersion, 1)
						if nd.nodeFragmentQueue.GetHeadRoot() == nil {
							if nd.DoneSync != nil {
								<-nd.DoneSync
							}
						}
						nd.requestInterval.Reset(RequestNodeInterval)
						nd.resetStateInterval.Reset(ResetNodeInterval)
						goto getAgain
					}
				}
				var nodes []*tpcmm.NodeInfo
				for _, i := range idNodes {
					nodes = append(nodes, i.V.(*tpcmm.NodeInfo))
				}

				//**sync nodes to local**+
				if err = nd.doNodeSync(nodes, nodeRoot); err != nil {
					goto getAgain
				}
			}
		} else if version < nd.curSyncVersion {
			nd.nodeFragmentQueue.dropHead()
		} else if version == curStateVersion+1 && nd.curSyncVersion == curStateVersion+1 {
			for {
				version, err := nd.nodeFragmentQueue.GetHeadPriority()
				if err != nil || version == 0 {
					goto getAgain
				}
				nodeRoot := nd.nodeFragmentQueue.GetHeadRoot()
				idNodes := nd.nodeFragmentQueue.PopHeadList()
				if idNodes == nil || len(idNodes) == 0 {
					curNodeRoot, err := nd.servant.GetNodeRoot()
					if err != nil {
						goto getAgain
					}
					if bytes.Equal(curNodeRoot, nd.remoteNodeRoot) {
						nd.nodeFragmentQueue.dropHead()
						atomic.AddUint64(&nd.curSyncVersion, 1)
						if nd.nodeFragmentQueue.GetHeadRoot() == nil {
							if nd.DoneSync != nil {
								<-nd.DoneSync
							}
						}
						nd.requestInterval.Reset(RequestNodeInterval)
						nd.resetStateInterval.Reset(ResetNodeInterval)
						goto getAgain
					}
				}
				var nodes []*tpcmm.NodeInfo
				for _, i := range idNodes {
					nodes = append(nodes, i.V.(*tpcmm.NodeInfo))
				}
				//**sync nodes to local**+
				if err = nd.doNodeSync(nodes, nodeRoot); err != nil {
					goto getAgain
				}
			}
		}

		select {
		case <-nd.chanQuitSync:
			return
		case <-nd.requestInterval.C:
			if err := nd.forceRequestNode(); err != nil {
				nd.resetCurLevel = true
				goto getAgain
			}
		case <-nd.resetStateInterval.C:
			nd.resetCurLevel = true
			if err := nd.forceRequestNode(); err != nil {
				goto getAgain
			}
			nd.resetStateInterval.Reset(ResetNodeInterval)
		default:
			continue
		}
	}
}

func (nd *NodeDownloader) doNodeSync(nodes []*tpcmm.NodeInfo, nodeRoot []byte) error {
	for _, node := range nodes {
		if !nd.servant.IsNodeExist(node.NodeID) {
			err := nd.servant.AddNode(node)
			if err != nil {
				nd.resetCurLevel = true
				return err
			}
		} else {
			err := nd.servant.UpdateWeight(node.NodeID, node.Weight)
			if err != nil {
				nd.resetCurLevel = true
				return err
			}
			err = nd.servant.UpdateDKGPartPubKey(node.NodeID, node.DKGPartPubKey)
			if err != nil {
				nd.resetCurLevel = true
				return err
			}
		}
	}
	nd.requestInterval.Reset(RequestNodeInterval)
	nd.resetStateInterval.Reset(ResetNodeInterval)
	curNodeRoot, err := nd.servant.GetNodeRoot()
	if err != nil {
		nd.resetCurLevel = true
		return err
	}
	if bytes.Equal(curNodeRoot, nodeRoot) {
		atomic.AddUint64(&nd.curSyncVersion, 1)
		nd.nodeFragmentQueue.dropHead()
		if nd.nodeFragmentQueue.GetHeadRoot() == nil {
			if nd.DoneSync != nil {
				<-nd.DoneSync
			}
		}
		return ErrDiffRoot
	}
	return nil
}

func (nd *NodeDownloader) forceRequestNode() error {
	if nd.DoneSync == nil {
		nd.requestInterval.Reset(RequestNodeInterval)
	}
	curNodeRoot, err := nd.servant.GetNodeRoot()
	if err != nil {
		nd.log.Errorf("get NodeRoot error", err)
		return err
	}
	if bytes.Equal(nd.remoteNodeRoot, curNodeRoot) {
		nd.requestInterval.Reset(RequestNodeInterval)
	}
	curStateVersion, err := nd.servant.StateLatestVersion()
	if err != nil {
		nd.log.Errorf("get StateLatestVersion error: ", err)
		return err
	}
	stateRequest := &StateRequest{
		StateVersion: curStateVersion,
	}
	data, _ := nd.marshaler.Marshal(stateRequest)
	syncMsg := SyncMessage{
		MsgType: SyncMessage_StateRequest,
		Data:    data,
	}
	syncData, _ := nd.marshaler.Marshal(syncMsg)
	nd.servant.Send(nd.ctx, protocol.AsyncSendProtocolID, MOD_NAME, syncData)
	nd.requestInterval.Reset(RequestNodeInterval)
	return nil
}

type ChainDownloader struct {
	*Downloader
	nodeID          string
	curSyncVersion  uint64
	remoteChainRoot []byte
	//savedVersions      *IdBitmap
	chanFetchChainData chan *FreeItem
	chainSyncQueue     *LockFreePriorityQueue
}

func NewChainDownloader(conf *SyncConfig, log tplog.Logger, marshaler codec.Marshaler, ctx context.Context, ledger ledger.Ledger, nodeID string) *ChainDownloader {
	downloader := &ChainDownloader{
		Downloader: NewDownloader(conf, log, marshaler, ctx, ledger),
		nodeID:     nodeID,
		//savedVersions:      new(IdBitmap),
		chanFetchChainData: make(chan *FreeItem, FetchChainChanSize),
		chainSyncQueue:     NewLKQueue(),
	}
	return downloader
}

func (cd *ChainDownloader) loopFetchChain() {
	for {
		select {
		case v := <-cd.chanFetchChainData:
			cd.chainSyncQueue.Push(v, v.value.(*ChainResponse).StateVersion)
		case <-cd.chanQuitSync:
			return
		default:
			continue
		}
	}
}
func (cd *ChainDownloader) loopDoSyncChain() {
	cd.requestInterval = time.NewTimer(RequestChainInterval)
	cd.resetStateInterval = time.NewTimer(ResetChainInterval)
	defer cd.requestInterval.Stop()
	defer cd.resetStateInterval.Stop()
	curStateVersion, err := cd.servant.StateLatestVersion()
	if err != nil {
		cd.log.Errorf("chain downLoader get StateLatestVersion error:", err)
	}
	cd.curSyncVersion = curStateVersion

	for {
	getAgain:
		version, err := cd.chainSyncQueue.GetHeadPriority()
		if err != nil || version == 0 {
			if cd.DoneSync != nil {
				<-cd.DoneSync
			}
			goto getAgain
		}
		curChainRoot, err := cd.servant.GetChainRoot()
		if err != nil {
			cd.log.Errorf("get StateRoot error:", err)
			goto getAgain
		}
		curStateVersion, err = cd.servant.StateLatestVersion()
		if err != nil {
			cd.log.Errorf("get latest version error:", err)
			goto getAgain
		}
		if cd.forceRequest {
			if err := cd.forceRequestChain(); err != nil {
				cd.resetCurLevel = true
				goto getAgain
			}
			cd.forceRequest = false
		}

		if version == curStateVersion && curStateVersion == cd.curSyncVersion {
			item := cd.chainSyncQueue.PopHead()
			chainInfo := item.value.(*ChainResponse)
			var msgBlock *tpchaintypes.Block
			err := cd.marshaler.Unmarshal(chainInfo.LatestBlock, &msgBlock)
			if err != nil {
				cd.log.Errorf("Unmarshal error:", err)
				goto getAgain
			}
			var msgBlockResult *tpchaintypes.BlockResult
			err = cd.marshaler.Unmarshal(chainInfo.LatestBlockResult, &msgBlockResult)
			if err != nil {
				cd.log.Errorf("Unmarshal error:", err)
				goto getAgain
			}
			msgHeight := msgBlock.Head.Height
			curHeight, err := cd.servant.GetLatestHeight()
			if err != nil {
				cd.log.Errorf("sync downloader chain get GetLatestHeight error:", err)
			}
			if msgHeight == curHeight+1 {
				if err = cd.doChainSync(msgBlock, msgBlockResult, chainInfo.ChainRoot, curChainRoot); err != nil {
					goto getAgain
				}
			} else if msgHeight == curHeight {
				curBlock, err := cd.servant.GetLatestBlock()
				if err != nil {
					cd.log.Errorf("sync downloader chain get GetLatestBlock error:", err)
					goto getAgain
				}
				if msgBlock.Head.Version > curBlock.Head.Version {
					err = cd.doChainSync(msgBlock, msgBlockResult, chainInfo.ChainRoot, curChainRoot)
					if err != nil {
						goto getAgain
					}
				}
			}
		} else if version == curStateVersion+1 && cd.curSyncVersion == curStateVersion+1 {
			item := cd.chainSyncQueue.PopHead()
			chainInfo := item.value.(*ChainResponse)
			var msgBlock *tpchaintypes.Block
			err = cd.marshaler.Unmarshal(chainInfo.LatestBlock, &msgBlock)
			if err != nil {
				cd.log.Errorf("Unmarshal error:", err)
				goto getAgain
			}
			var msgBlockResult *tpchaintypes.BlockResult
			err = cd.marshaler.Unmarshal(chainInfo.LatestBlockResult, &msgBlockResult)
			if err != nil {
				cd.log.Errorf("Unmarshal error:", err)
				goto getAgain
			}
			cd.doChainSync(msgBlock, msgBlockResult, chainInfo.ChainRoot, curChainRoot)
		}

		select {
		case <-cd.chanQuitSync:
			return
		case <-cd.requestInterval.C:
			if err := cd.forceRequestChain(); err != nil {
				cd.resetCurLevel = true
				goto getAgain
			}
		case <-cd.resetStateInterval.C:
			cd.resetCurLevel = true
			if err := cd.forceRequestChain(); err != nil {
				goto getAgain
			}
			cd.resetStateInterval.Reset(ResetChainInterval)
		default:
			continue
		}
	}
}

func (cd *ChainDownloader) doChainSync(msgBlock *tpchaintypes.Block, msgBlockResult *tpchaintypes.BlockResult, chainRoot, curChainRoot []byte) error {
	err := cd.servant.SetLatestBlock(msgBlock)
	if err != nil {
		cd.resetCurLevel = true
		return err
	}
	if err := cd.servant.SetLatestBlockResult(msgBlockResult); err != nil {
		cd.resetCurLevel = true
		return err
	}

	if !bytes.Equal(chainRoot, curChainRoot) {
		cd.resetCurLevel = true
		return ErrDiffRoot
	}
	atomic.AddUint64(&cd.curSyncVersion, 1)
	cd.requestInterval.Reset(RequestChainInterval)
	cd.resetStateInterval.Reset(ResetChainInterval)
	return nil
}

func (cd *ChainDownloader) forceRequestChain() error {
	if cd.DoneSync == nil {
		cd.requestInterval.Reset(RequestChainInterval)
	}
	curChainRoot, err := cd.servant.GetChainRoot()
	if err != nil {
		cd.log.Errorf("get ChainRoot error", err)
		return err
	}
	if bytes.Equal(cd.remoteChainRoot, curChainRoot) {
		cd.requestInterval.Reset(RequestChainInterval)
	}
	curStateRoot, err := cd.servant.StateRoot()
	if err != nil {
		cd.log.Errorf("get StateRoot error: ", err)
		return err
	}
	curStateVersion, err := cd.servant.StateLatestVersion()
	if err != nil {
		cd.log.Errorf("get StateLatestVersion error: ", err)
		return err
	}
	curChainID := cd.servant.ChainID()
	curNetworkType := cd.servant.NetworkType()
	dataNetworkType, _ := cd.marshaler.Marshal(curNetworkType)
	curLatestHeight, err := cd.servant.GetLatestHeight()
	if err != nil {
		cd.log.Errorf("get LatestHeight error: ", err)
		return err
	}
	curLatestBlock, err := cd.servant.GetLatestBlock()
	if err != nil {
		cd.log.Errorf("get LatestBlock error: ", err)
		return err
	}
	dataLatestBlock, _ := cd.marshaler.Marshal(curLatestBlock)
	curLatestResult, err := cd.servant.GetLatestBlockResult()
	if err != nil {
		cd.log.Errorf("get LatestBlockResult error: ", err)
		return err
	}
	dataLatestBlockResult, _ := cd.marshaler.Marshal(curLatestResult)

	chainRequest := &ChainResponse{
		StateRoot:         curStateRoot,
		StateVersion:      curStateVersion,
		ChainRoot:         curChainRoot,
		ChainID:           string(curChainID),
		NetworkType:       dataNetworkType,
		LatestHeight:      curLatestHeight,
		LatestBlock:       dataLatestBlock,
		LatestBlockResult: dataLatestBlockResult,
	}
	data, _ := cd.marshaler.Marshal(chainRequest)
	zipData, _ := CompressBytes(data, GZIP)
	syncMsgChain := SyncMessage{
		MsgType: SyncMessage_ChainResponse,
		Data:    zipData,
	}
	syncData, _ := cd.marshaler.Marshal(&syncMsgChain)
	cd.servant.Send(cd.ctx, protocol.AsyncSendProtocolID, MOD_NAME, syncData)
	cd.requestInterval.Reset(RequestChainInterval)
	return nil
}

type AccountDownloader struct {
	*Downloader
	curSyncVersion           uint64
	remoteAccountRoot        []byte
	chanFetchAccountFragment chan *AccountResponse
	accFragmentQueue         *LockFreeSetQueue
}

func NewAccountDownloader(conf *SyncConfig, log tplog.Logger, marshaler codec.Marshaler, ctx context.Context, ledger ledger.Ledger) *AccountDownloader {
	downloader := &AccountDownloader{
		Downloader:               NewDownloader(conf, log, marshaler, ctx, ledger),
		chanFetchAccountFragment: make(chan *AccountResponse, FetchAccountChanSize),
		accFragmentQueue:         NewLKQueueSet(),
	}
	return downloader
}

func (ad *AccountDownloader) loopFetchAccounts() {
	for {
		select {
		case v := <-ad.chanFetchAccountFragment:
			tmpItem := &LKQueueSet{
				Map:       make(map[interface{}]struct{}, 0),
				Priority:  v.StateVersion,
				Root:      v.AccountRoot,
				ListBytes: v.AccountsData,
			}
			var list []*account.Account
			err := ad.marshaler.Unmarshal(tmpItem.ListBytes, list)
			if err != nil {
				ad.log.Errorf("Unmarshal list bytes err", err)
				return
			}
			var addrAccount *KeyValueItem
			for _, account := range list {
				addrAccount.K = account.Addr
				addrAccount.V = account
				tmpItem.List = append(tmpItem.List, addrAccount)
			}
			ad.accFragmentQueue.Push(tmpItem, v.StateVersion)

		case <-ad.chanQuitSync:
			return
		default:
			continue
		}
	}
}
func (ad *AccountDownloader) loopDoSyncAccounts() {
	ad.requestInterval = time.NewTimer(RequestAccountInterval)
	ad.resetStateInterval = time.NewTimer(ResetAccountStateInterval)
	defer ad.requestInterval.Stop()
	defer ad.resetStateInterval.Stop()
	curStateVersion, err := ad.servant.StateLatestVersion()
	if err != nil {
		ad.log.Errorf("account downLoader get StateLatestVersion error:", err)
	}
	ad.curSyncVersion = curStateVersion

	for {
	getAgain:
		version, err := ad.accFragmentQueue.GetHeadPriority()
		if err != nil || version == 0 {
			if ad.DoneSync != nil {
				<-ad.DoneSync
			}
			goto getAgain
		}
		curStateVersion, err = ad.servant.StateLatestVersion()
		if err != nil || version == 0 {
			goto getAgain
		}

		if ad.forceRequest {
			if err := ad.forceRequestAccount(); err != nil {
				ad.resetCurLevel = true
				goto getAgain
			}
			ad.forceRequest = false
		}

		if version == curStateVersion && curStateVersion == ad.curSyncVersion {
			for {
				version, err = ad.accFragmentQueue.GetHeadPriority()
				if err != nil || version == 0 {
					goto getAgain
				}
				if version != ad.curSyncVersion {
					goto getAgain
				}
				accRoot := ad.accFragmentQueue.GetHeadRoot()
				accs := ad.accFragmentQueue.PopHeadList()
				if accs == nil || len(accs) == 0 {
					curAccountRoot, err := ad.servant.GetAccountRoot()
					if err != nil {
						goto getAgain
					}
					if bytes.Equal(curAccountRoot, ad.remoteAccountRoot) {
						ad.accFragmentQueue.dropHead()
						atomic.AddUint64(&ad.curSyncVersion, 1)
						if ad.accFragmentQueue.GetHeadRoot() == nil {
							if ad.DoneSync != nil {
								<-ad.DoneSync
							}
						}
						goto getAgain
					}
				}
				var accounts []*account.Account
				for _, i := range accs {
					accounts = append(accounts, i.V.(*account.Account))
				}
				if err = ad.doAccountSync(accounts, accRoot); err != nil {
					goto getAgain
				}
			}
		} else if version < ad.curSyncVersion {
			ad.accFragmentQueue.dropHead()
		} else if version == curStateVersion+1 && ad.curSyncVersion == curStateVersion+1 {
			for {
				version, err = ad.accFragmentQueue.GetHeadPriority()
				if err != nil || version == 0 {
					goto getAgain
				}
				if version != ad.curSyncVersion {
					goto getAgain
				}
				accRoot := ad.accFragmentQueue.GetHeadRoot()
				accs := ad.accFragmentQueue.PopHeadList()
				if accs == nil || len(accs) == 0 {
					curAccountRoot, err := ad.servant.GetAccountRoot()
					if err != nil {
						goto getAgain
					}
					if bytes.Equal(curAccountRoot, ad.remoteAccountRoot) {
						ad.accFragmentQueue.dropHead()
						atomic.AddUint64(&ad.curSyncVersion, 1)
						if ad.accFragmentQueue.GetHeadRoot() == nil {
							if ad.DoneSync != nil {
								<-ad.DoneSync
							}
						}
						goto getAgain
					}
				}
				var accounts []*account.Account
				for _, i := range accs {
					accounts = append(accounts, i.V.(*account.Account))
				}
				if err = ad.doAccountSync(accounts, accRoot); err != nil {
					goto getAgain
				}
			}
		}

		select {
		case <-ad.chanQuitSync:
			return
		case <-ad.requestInterval.C:
			if err := ad.forceRequestAccount(); err != nil {
				ad.resetCurLevel = true
				goto getAgain
			}
		case <-ad.resetStateInterval.C:
			ad.resetCurLevel = true
			if err := ad.forceRequestAccount(); err != nil {
				goto getAgain
			}
		default:
			continue
		}
	}
}

func (ad *AccountDownloader) doAccountSync(accounts []*account.Account, accRoot []byte) error {

	//**sync Accounts to local**
	for _, account := range accounts {
		if !ad.servant.IsAccountExist(account.Addr) {
			err := ad.servant.AddAccount(account)
			if err != nil {
				ad.resetCurLevel = true
				return err
			}
		} else {
			err := ad.servant.UpdateAccount(account)
			if err != nil {
				ad.resetCurLevel = true
				return err
			}
		}
	}
	ad.requestInterval.Reset(RequestAccountInterval)
	ad.resetStateInterval.Reset(ResetAccountStateInterval)
	curAccountRoot, err := ad.servant.GetAccountRoot()
	if err != nil {
		ad.resetCurLevel = true
		return err
	}
	if bytes.Equal(curAccountRoot, accRoot) {
		atomic.AddUint64(&ad.curSyncVersion, 1)
		ad.accFragmentQueue.dropHead()
		if ad.accFragmentQueue.GetHeadRoot() == nil {
			if ad.DoneSync != nil {
				<-ad.DoneSync
			}
		}
		return ErrDiffRoot
	}
	return nil
}

func (ad *AccountDownloader) forceRequestAccount() error {
	if ad.DoneSync == nil {
		ad.requestInterval.Reset(RequestAccountInterval)
	}
	curAccountRoot, err := ad.servant.GetAccountRoot()
	if err != nil {
		ad.log.Errorf("get AccountRoot error", err)
		return err
	}
	if bytes.Equal(ad.remoteAccountRoot, curAccountRoot) {
		ad.requestInterval.Reset(RequestAccountInterval)
	}

	curStateVersion, err := ad.servant.StateLatestVersion()
	if err != nil {
		ad.log.Errorf("get StateLatestVersion error: ", err)
		return err
	}
	stateRequest := &StateRequest{
		StateVersion: curStateVersion,
	}
	data, _ := ad.marshaler.Marshal(stateRequest)
	syncMsg := SyncMessage{
		MsgType: SyncMessage_StateRequest,
		Data:    data,
	}
	syncData, _ := ad.marshaler.Marshal(&syncMsg)
	ad.servant.Send(ad.ctx, protocol.AsyncSendProtocolID, MOD_NAME, syncData)
	ad.requestInterval.Reset(RequestAccountInterval)
	return nil
}
