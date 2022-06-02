package sync

import (
	"context"
	tpchaintypes "github.com/TopiaNetwork/topia/chain/types"
	"github.com/TopiaNetwork/topia/codec"
	"github.com/TopiaNetwork/topia/common"
	tplog "github.com/TopiaNetwork/topia/log"
	"github.com/TopiaNetwork/topia/network/protocol"
	"github.com/TopiaNetwork/topia/state/node"
	"time"
)

type Downloader struct {
	log             tplog.Logger
	marshaler       codec.Marshaler
	servant         SyncServant
	ctx             context.Context
	chanQuitSync    chan struct{}
	requestInterval *time.Timer
}

func NewDownloader(log tplog.Logger, marshaler codec.Marshaler, ctx context.Context) *Downloader {
	downloader := &Downloader{
		log:          log,
		marshaler:    marshaler,
		ctx:          ctx,
		chanQuitSync: make(chan struct{}),
	}
	return downloader
}

type BlockDownloader struct {
	*Downloader
	savedBlockHeights *common.IdBitmap //save block height for block saved in queue
	chanFetchBlock    chan *tpchaintypes.Block
	blockSyncQueue    *common.LockFreePriorityQueue //look free height sorted queue for blocks

}

func NewBlockDownloader(log tplog.Logger, marshaler codec.Marshaler, ctx context.Context) *BlockDownloader {
	downloader := &BlockDownloader{
		Downloader:        NewDownloader(log, marshaler, ctx),
		savedBlockHeights: new(common.IdBitmap),
		blockSyncQueue:    common.NewLKQueue(),
	}
	return downloader
}

func (bd *BlockDownloader) loopFetchBlocks() {

	for {
		select {
		case v := <-bd.chanFetchBlock:
			if bd.savedBlockHeights.Add(v.Head.Height) {
				bd.blockSyncQueue.Push(v, v.Head.Height)
			}
		case <-bd.chanQuitSync:
			return
		default:
			continue
		}
	}
}

func (bd *BlockDownloader) loopDoBlockSync() {
	bd.requestInterval = time.NewTimer(RequestBlockInterval)
	defer bd.requestInterval.Stop()
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
		if curBlockHeight == height {
			blockInterface := bd.blockSyncQueue.PopHead()
			block := blockInterface.(*tpchaintypes.Block)
			if block.Head.Height > height {
				bd.blockSyncQueue.Push(block, block.Head.Height)
			} else if block.Head.Height < height {
				goto getAgain
			}
			bd.savedBlockHeights.Sub(block.Head.Height)
			BlockExec := func() {
				//sync block to local
			}
			BlockExec()
			bd.requestInterval.Reset(RequestBlockInterval)
		} else if curBlockHeight > height {
			bd.blockSyncQueue.PopHead()
			goto getAgain
		}

		select {
		case <-bd.chanQuitSync:
			return
		case <-bd.requestInterval.C:

			blockRequest := &BlockRequest{
				Height: curBlockHeight,
			}
			data, _ := bd.marshaler.Marshal(blockRequest)
			syncMsg := SyncMessage{
				MsgType: SyncMessage_BlockRequest,
				Data:    data,
			}

			syncData, _ := bd.marshaler.Marshal(&syncMsg)
			bd.servant.Send(bd.ctx, protocol.SyncProtocolID_Block, MOD_NAME, syncData)
			bd.savedBlockHeights.Sub(height)
			bd.requestInterval.Reset(RequestBlockInterval)
		default:
			continue
		}
	}
}

type EpochDownloader struct {
	*Downloader
	savedEpochs        *common.IdBitmap //save epochs for epochInfos saved in queue
	chanFetchEpochInfo chan *common.EpochInfo
	epochInfoSyncQueue *common.LockFreePriorityQueue //look free height sorted queue for EpochInfos
}

func NewEpochDownloader(log tplog.Logger, marshaler codec.Marshaler, ctx context.Context) *EpochDownloader {
	downloader := &EpochDownloader{
		Downloader:         NewDownloader(log, marshaler, ctx),
		savedEpochs:        new(common.IdBitmap),
		epochInfoSyncQueue: common.NewLKQueue(),
	}
	return downloader
}

func (ed *EpochDownloader) loopFetchEpochs() {
	for {
		select {
		case v := <-ed.chanFetchEpochInfo:
			if ed.savedEpochs.Add(v.Epoch) {
				ed.epochInfoSyncQueue.Push(v, v.Epoch)
			}
		case <-ed.chanQuitSync:
			return
		default:
			continue
		}
	}
}

func (ed *EpochDownloader) loopDoEpochSync() {
	ed.requestInterval = time.NewTimer(RequestEpochInterval)
	defer ed.requestInterval.Stop()
	for {
	getAgain:
		epoch, err := ed.epochInfoSyncQueue.GetHeadPriority()
		if err != nil || epoch == 0 {
			goto getAgain
		}
		curEpochInfo, err := ed.servant.GetLatestEpoch()
		curEpoch := curEpochInfo.Epoch
		if curEpoch == epoch {
			epochInterface := ed.epochInfoSyncQueue.PopHead()
			epochInfo := epochInterface.(*common.EpochInfo)
			if epochInfo.Epoch > epoch {
				ed.epochInfoSyncQueue.Push(curEpochInfo, curEpoch)
			} else if epochInfo.Epoch < epoch {
				goto getAgain
			}
			ed.savedEpochs.Sub(curEpoch)

			EpochExec := func() {
				//sync EpochInfo to local
			}
			EpochExec()
			ed.requestInterval.Reset(RequestEpochInterval)
		} else if curEpoch > epoch {
			ed.epochInfoSyncQueue.PopHead()
			goto getAgain
		}

		select {
		case <-ed.chanQuitSync:
			return
		case <-ed.requestInterval.C:

			epochRequest := &EpochRequest{
				Epoch: curEpoch,
			}
			data, _ := ed.marshaler.Marshal(epochRequest)
			syncMsg := SyncMessage{
				MsgType: SyncMessage_EpochRequest,
				Data:    data,
			}

			syncData, _ := ed.marshaler.Marshal(&syncMsg)
			ed.servant.Send(ed.ctx, protocol.SyncProtocolId_Epoch, MOD_NAME, syncData)
			ed.savedEpochs.Sub(curEpoch)
			ed.requestInterval.Reset(RequestEpochInterval)
		default:
			continue
		}
	}
}

type NodeStateDownloader struct {
	*Downloader
	savedVersions      *common.IdBitmap //save NodeState versions for nodeState saved in queue
	chanFetchNodeState chan node.NodeState
	NodeStateSyncQueue *common.LockFreePriorityQueue //look free height sorted queue for EpochInfos
}

func NewNodeStateDownloader(log tplog.Logger, marshaler codec.Marshaler, ctx context.Context) *NodeStateDownloader {
	downloader := &NodeStateDownloader{
		Downloader:         NewDownloader(log, marshaler, ctx),
		savedVersions:      new(common.IdBitmap),
		NodeStateSyncQueue: common.NewLKQueue(),
	}
	return downloader
}

func (nd *NodeStateDownloader) loopFetchNodeState() {
	for {
		select {
		case v := <-nd.chanFetchNodeState:
			version, _ := v.GetNodeLatestStateVersion()
			if nd.savedVersions.Add(version) {
				nd.NodeStateSyncQueue.Push(v, version)
			}
		case <-nd.chanQuitSync:
			return
		default:
			continue
		}
	}
}

func (nd *NodeStateDownloader) loopDoNodeStateSync() {
	nd.requestInterval = time.NewTimer(RequestNodeStateInterval)
	defer nd.requestInterval.Stop()
	for {
	getAgain:
		version, err := nd.NodeStateSyncQueue.GetHeadPriority()
		if err != nil || version == 0 {
			goto getAgain
		}
		curNodeStateVersion, err := nd.servant.GetNodeLatestStateVersion()
		if curNodeStateVersion == version {
			nodeStateInterface := nd.NodeStateSyncQueue.PopHead()
			nodeState := nodeStateInterface.(node.NodeState)
			nodeStateVersion, _ := nodeState.GetNodeLatestStateVersion()
			if nodeStateVersion > version {
				nd.NodeStateSyncQueue.Push(nodeState, nodeStateVersion)
			} else if nodeStateVersion < version {
				goto getAgain
			}
			nd.savedVersions.Sub(curNodeStateVersion)

			nodeStateExec := func() {
				//sync block to local
			}
			nodeStateExec()
			nd.requestInterval.Reset(RequestNodeStateInterval)
		} else if curNodeStateVersion > version {
			nd.NodeStateSyncQueue.PopHead()
			goto getAgain
		}

		select {
		case <-nd.chanQuitSync:
			return
		case <-nd.requestInterval.C:

			nodeStateRequest := &NodeRequest{
				StateVersion:         curNodeStateVersion,
				XXX_NoUnkeyedLiteral: struct{}{},
				XXX_unrecognized:     nil,
				XXX_sizecache:        0,
			}
			data, _ := nd.marshaler.Marshal(nodeStateRequest)
			syncMsg := SyncMessage{
				MsgType: SyncMessage_NodeStateRequest,
				Data:    data,
			}

			syncData, _ := nd.marshaler.Marshal(&syncMsg)
			nd.servant.Send(nd.ctx, protocol.SyncProtocolID_Block, MOD_NAME, syncData)
			nd.savedVersions.Sub(curNodeStateVersion)
			nd.requestInterval.Reset(RequestNodeStateInterval)
		default:
			continue
		}
	}
}
