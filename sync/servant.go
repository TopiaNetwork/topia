package sync

import (
	"context"
	"errors"
	"sync"
	"time"

	tpacc "github.com/TopiaNetwork/topia/account"
	tpchaintypes "github.com/TopiaNetwork/topia/chain/types"
	tpcmm "github.com/TopiaNetwork/topia/common"
	tpcrtypes "github.com/TopiaNetwork/topia/crypt/types"
	"github.com/TopiaNetwork/topia/ledger"
	tplgblock "github.com/TopiaNetwork/topia/ledger/block"
	"github.com/TopiaNetwork/topia/network"
	tpnetcmn "github.com/TopiaNetwork/topia/network/common"
	"github.com/TopiaNetwork/topia/network/message"
	"github.com/TopiaNetwork/topia/service"
	"github.com/TopiaNetwork/topia/state"
	"github.com/TopiaNetwork/topia/state/account"
	"github.com/TopiaNetwork/topia/state/chain"
	"github.com/TopiaNetwork/topia/state/epoch"
	node2 "github.com/TopiaNetwork/topia/state/node"
	txbasic "github.com/TopiaNetwork/topia/transaction/basic"
)

type SyncServant interface {
	GetLatestHeight() (uint64, error)

	GetLatestBlockHash() (tpchaintypes.BlockHash, error)

	GetBlockByNumber(blockNum tpchaintypes.BlockNum) (*tpchaintypes.Block, error)

	GetLatestBlock() (*tpchaintypes.Block, error)

	SetLatestBlock(block *tpchaintypes.Block) error

	SetLatestBlockResult(blockResult *tpchaintypes.BlockResult) error

	StateRoot() ([]byte, error)

	StateLatestVersion() (uint64, error)

	GetAccountRoot() ([]byte, error) //account state root

	GetAllAccountAddress() ([]tpcrtypes.Address, error)

	GetNonce(addr tpcrtypes.Address) (uint64, error)

	IsAccountExist(addr tpcrtypes.Address) bool

	AddAccount(acc *tpacc.Account) error

	UpdateAccount(account *tpacc.Account) error

	GetChainRoot() ([]byte, error) //chain state root

	GetLatestBlockResult() (*tpchaintypes.BlockResult, error)

	BlockResultForTxsResult(result []*txbasic.TransactionResult) (*tpchaintypes.BlockResult, error)

	ChainID() tpchaintypes.ChainID

	GetLatestEpoch() (*tpcmm.EpochInfo, error)

	SetLatestEpoch(epoch *tpcmm.EpochInfo) error

	GetEpochRoot() ([]byte, error) //epoch state root

	GetNodeRoot() ([]byte, error) //node state root
	IsNodeExist(nodeID string) bool
	AddNode(nodeInfo *tpcmm.NodeInfo) error
	UpdateWeight(nodeID string, weight uint64) error
	UpdateDKGPartPubKey(nodeID string, pubKey string) error

	NetworkType() tpcmm.NetworkType

	GetAllConsensusNodeIDs() ([]string, error)

	Send(ctx context.Context, protocolID string, moduleName string, data []byte) error

	SendWithResponse(ctx context.Context, protocolID string, moduleName string, data []byte) ([]message.SendResponse, error)

	PubSubScores() []tpnetcmn.PubsubScore

	SetNodeScore(nodeID string, nodeScore *nodeScore)

	RemoveNodeScore(nodeID string)

	GetNodeScore(nodeID string) float64

	ReNewNodeScores()

	GetBlockStore() tplgblock.BlockStore

	IsBadBlockExist(hash tpchaintypes.BlockHash) bool

	AddBadBlock(hash tpchaintypes.BlockHash, height uint64) error

	RemoveBadBlock(hash tpchaintypes.BlockHash)
}

func NewSyncServant(stateQueryService service.StateQueryService, blockService service.BlockService) SyncServant {
	syncservant := &syncServant{
		StateQueryService: stateQueryService,
		BlockService:      blockService,
		nodeScores:        NewNodeScores(),
		badBlocks:         NewBlackBlocks(),
	}
	return syncservant
}

type syncServant struct {
	service.StateQueryService
	service.BlockService
	network.Network
	ledger.Ledger
	comp         state.CompositionState
	accountState account.AccountState
	chainState   chain.ChainState
	epochState   epoch.EpochState
	nodeState    node2.NodeState
	nodeScores   *nodeScores
	badBlocks    *BadBlocks
}

func (syncServ *syncServant) GetLatestHeight() (uint64, error) {
	curBlock, err := syncServ.GetLatestBlock()
	if err != nil {
		return 0, err
	}
	curHeight := curBlock.Head.Height
	return curHeight, nil
}

func (syncServ *syncServant) GetLatestBlockHash() (tpchaintypes.BlockHash, error) {
	block, err := syncServ.GetLatestBlock()
	if err != nil {
		return "", err
	}
	blockHash, err := block.BlockHash()
	if err != nil {
		return "", err
	}
	return blockHash, nil
}

func (syncServ *syncServant) GetNodeRoot() ([]byte, error) {
	return syncServ.nodeState.GetNodeRoot()
}

func (syncServ *syncServant) IsAccountExist(addr tpcrtypes.Address) bool {
	return syncServ.accountState.IsAccountExist(addr)
}

func (syncServ *syncServant) GetAccountRoot() ([]byte, error) {
	return syncServ.accountState.GetAccountRoot()
}

func (syncServ *syncServant) AddAccount(acc *tpacc.Account) error {
	return syncServ.accountState.AddAccount(acc)
}

func (syncServ *syncServant) UpdateAccount(account *tpacc.Account) error {
	return syncServ.accountState.UpdateAccount(account)
}

func (syncServ *syncServant) GetAllAccountAddress() ([]tpcrtypes.Address, error) {
	accounts, err := syncServ.accountState.GetAllAccounts()
	if err != nil {
		return nil, err
	}
	var addrs []tpcrtypes.Address
	for _, account := range accounts {
		addrs = append(addrs, account.Addr)
	}
	return addrs, nil
}
func (syncServ *syncServant) ResetAccountLatestState() error {
	panic("implement me")
}
func (syncServ *syncServant) ResetNodeLatestState() error {
	panic("implement me")
}

func (syncServ *syncServant) SetLatestBlock(block *tpchaintypes.Block) error {
	return syncServ.chainState.SetLatestBlock(block)
}
func (syncServ *syncServant) SetLatestBlockResult(blockResult *tpchaintypes.BlockResult) error {
	return syncServ.chainState.SetLatestBlockResult(blockResult)
}

func (syncServ *syncServant) GetChainRoot() ([]byte, error) {
	return syncServ.chainState.GetChainRoot()
}
func (syncServ *syncServant) GetEpochRoot() ([]byte, error) {
	return syncServ.epochState.GetEpochRoot()
}
func (syncServ *syncServant) SetLatestEpoch(epoch *tpcmm.EpochInfo) error {
	return syncServ.epochState.SetLatestEpoch(epoch)
}

func (syncServ *syncServant) IsNodeExist(nodeID string) bool {
	return syncServ.nodeState.IsNodeExist(nodeID)
}
func (syncServ *syncServant) AddNode(nodeInfo *tpcmm.NodeInfo) error {
	return syncServ.nodeState.AddNode(nodeInfo)
}
func (syncServ *syncServant) UpdateWeight(nodeID string, weight uint64) error {
	return syncServ.nodeState.UpdateWeight(nodeID, weight)
}

func (syncServ *syncServant) UpdateDKGPartPubKey(nodeID string, pubKey string) error {
	return syncServ.nodeState.UpdateDKGPartPubKey(nodeID, pubKey)
}

func (syncServ *syncServant) BlockResultForTxsResult(result []*txbasic.TransactionResult) (*tpchaintypes.BlockResult, error) {
	return syncServ.GetLatestBlockResult()
}

func (syncServ *syncServant) SetNodeScore(nodeID string, nodeScore *nodeScore) {
	syncServ.nodeScores.SetNodeScore(nodeID, nodeScore)
}

func (syncServ *syncServant) RemoveNodeScore(nodeID string) {
	syncServ.nodeScores.RemoveNodeScore(nodeID)
}
func (syncServ *syncServant) ReNewNodeScores() {
	pubSubScores := syncServ.PubSubScores()
	for _, pubSubScore := range pubSubScores {
		tmpNodeScore := &nodeScore{
			score:    pubSubScore.Score.Score,
			lastTime: time.Now(),
		}
		syncServ.SetNodeScore(pubSubScore.ID, tmpNodeScore)
	}
}

func (syncServ *syncServant) GetNodeScore(nodeID string) float64 {
	return syncServ.nodeScores.GetNodeScore(nodeID)
}

func (syncServ *syncServant) IsBadBlockExist(hash tpchaintypes.BlockHash) bool {
	return syncServ.badBlocks.IsBadBlockExist(hash)
}

func (syncServ *syncServant) AddBadBlock(hash tpchaintypes.BlockHash, height uint64) error {
	return syncServ.badBlocks.AddBadBlock(hash, height)
}

func (syncServ *syncServant) RemoveBadBlock(hash tpchaintypes.BlockHash) {
	syncServ.badBlocks.RemoveBadBlock(hash)
}

type BadBlocks struct {
	mu         sync.RWMutex
	HashHeight map[tpchaintypes.BlockHash]uint64
}

func NewBlackBlocks() *BadBlocks {
	bb := &BadBlocks{
		HashHeight: make(map[tpchaintypes.BlockHash]uint64, 0),
	}
	return bb
}

func (bb *BadBlocks) IsBadBlockExist(hash tpchaintypes.BlockHash) bool {
	bb.mu.RLock()
	defer bb.mu.RUnlock()
	_, ok := bb.HashHeight[hash]
	return ok
}

func (bb *BadBlocks) AddBadBlock(hash tpchaintypes.BlockHash, height uint64) error {
	bb.mu.RLock()
	defer bb.mu.RUnlock()
	if _, ok := bb.HashHeight[hash]; !ok {
		bb.HashHeight[hash] = height
	} else {
		return errors.New("BadBlock is exist")
	}
	return nil
}

func (bb *BadBlocks) RemoveBadBlock(hash tpchaintypes.BlockHash) {
	bb.mu.RLock()
	defer bb.mu.RUnlock()
	delete(bb.HashHeight, hash)
}

type nodeScore struct {
	score    float64
	lastTime time.Time
}

type nodeScores struct {
	mu          sync.RWMutex
	nodeIDScore map[string]*nodeScore
	maxScore    float64
}

func NewNodeScores() *nodeScores {
	ns := &nodeScores{
		nodeIDScore: make(map[string]*nodeScore, 0),
		maxScore:    0,
	}
	return ns
}

func (ns *nodeScores) SetNodeScore(nodeID string, nodeScore *nodeScore) {
	ns.mu.Lock()
	defer ns.mu.Unlock()
	if _, ok := ns.nodeIDScore[nodeID]; !ok {
		ns.nodeIDScore[nodeID] = nodeScore
		if nodeScore.score > ns.maxScore {
			ns.maxScore = nodeScore.score
		}
	} else {
		ns.nodeIDScore[nodeID] = nodeScore
	}
}
func (ns *nodeScores) RemoveNodeScore(nodeID string) {
	ns.mu.Lock()
	defer ns.mu.Unlock()
	delete(ns.nodeIDScore, nodeID)
}
func (ns *nodeScores) GetNodeScore(nodeID string) float64 {
	ns.mu.RLock()
	defer ns.mu.RUnlock()
	return ns.nodeIDScore[nodeID].score
}
