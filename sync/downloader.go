package sync

import (
	"context"
	tpchaintypes "github.com/TopiaNetwork/topia/chain/types"
	"github.com/TopiaNetwork/topia/codec"
	"github.com/TopiaNetwork/topia/common"
	tplog "github.com/TopiaNetwork/topia/log"
	"github.com/TopiaNetwork/topia/network/protocol"
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
	savedBlockHeights *IdBitmap //save block height for block saved in queue
	chanFetchBlock    chan *tpchaintypes.Block
	blockSyncQueue    *LockFreePriorityQueue //look free height sorted queue for blocks

}

func NewBlockDownloader(log tplog.Logger, marshaler codec.Marshaler, ctx context.Context) *BlockDownloader {
	downloader := &BlockDownloader{
		Downloader:        NewDownloader(log, marshaler, ctx),
		savedBlockHeights: new(IdBitmap),
		blockSyncQueue:    NewLKQueue(),
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
	savedEpochs        *IdBitmap //save epochs for epochInfos saved in queue
	chanFetchEpochInfo chan *common.EpochInfo
	epochInfoSyncQueue *LockFreePriorityQueue //look free height sorted queue for EpochInfos
}

func NewEpochDownloader(log tplog.Logger, marshaler codec.Marshaler, ctx context.Context) *EpochDownloader {
	downloader := &EpochDownloader{
		Downloader:         NewDownloader(log, marshaler, ctx),
		savedEpochs:        new(IdBitmap),
		epochInfoSyncQueue: NewLKQueue(),
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
			curStateVersion, err := ed.servant.StateLatestVersion()
			if err != nil {
				ed.log.Errorf("get StateLatestVersion error: ", err)
			}
			curEpochRoot, err := ed.servant.GetRoundStateRoot()
			if err != nil {
				ed.log.Errorf("get GetRoundStateRoot error: ", err)
			}
			epochRequest := &StateRequest{
				StateVersion: curStateVersion,
				EpochRoot:    curEpochRoot,
			}
			data, _ := ed.marshaler.Marshal(epochRequest)
			syncMsg := SyncMessage{
				MsgType: SyncMessage_StateRequest,
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

type StateDownloader struct {
	*Downloader
	savedVersions      *IdBitmap
	chanFetchStateData chan []byte
	stateSyncQueue     *LockFreePriorityQueue
}

func (sd *StateDownloader) loopFetchStates() {

}
func (sd *StateDownloader) loopDoStateSync() {

}

type NodeDownloader struct {
	*Downloader
	savedVersions     *IdBitmap
	chanFetchNodeData chan []byte
	nodeSyncQueue     *LockFreePriorityQueue
}

func (nd *NodeDownloader) loopFetchNodes() {

}
func (nd *NodeDownloader) loopDoNodeSync() {

}

type ChainDownloader struct {
	*Downloader
	savedVersions      *IdBitmap
	chanFetchChainData chan []byte
	chainSyncQueue     *LockFreePriorityQueue
}

func (cd *ChainDownloader) loopFetchChains() {

}
func (cd *ChainDownloader) loopDoChainSync() {

}

type AccountDownloader struct {
	*Downloader
	savedVersions        *IdBitmap
	chanFetchAccountData chan []byte
	accountSyncQueue     *LockFreePriorityQueue
}

func (ad *AccountDownloader) loopFetchAccounts() {

}
func (ad *AccountDownloader) loopDoAccountSync() {

}
