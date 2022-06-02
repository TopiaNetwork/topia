package sync

import (
	"context"
	tpchaintypes "github.com/TopiaNetwork/topia/chain/types"
	tpcmm "github.com/TopiaNetwork/topia/common"
	"github.com/TopiaNetwork/topia/network"
	"github.com/TopiaNetwork/topia/service"
	"github.com/TopiaNetwork/topia/state"
	"github.com/TopiaNetwork/topia/state/account"
	"github.com/TopiaNetwork/topia/state/chain"
	"github.com/TopiaNetwork/topia/state/epoch"
	node2 "github.com/TopiaNetwork/topia/state/node"
)

type SyncServant interface {
	GetLatestHeight() (uint64, error)
	GetLatestBlockHash() (tpchaintypes.BlockHash, error)
	GetBlockByNumber(blockNum tpchaintypes.BlockNum) (*tpchaintypes.Block, error)
	GetLatestBlock() (*tpchaintypes.Block, error)
	SetBlock(block *tpchaintypes.Block) error

	StateRoot() ([]byte, error)

	StateLatestVersion() (uint64, error)

	GetAccountRoot() ([]byte, error) //account state root

	GetChainRoot() ([]byte, error) //chain state root

	GetLatestEpoch() (*tpcmm.EpochInfo, error)
	SetLatestEpoch(epoch *tpcmm.EpochInfo) error
	GetRoundStateRoot() ([]byte, error) //epoch state root

	GetNodeStateRoot() ([]byte, error) //node state root

	Send(ctx context.Context, protocolID string, moduleName string, data []byte) error
}

func NewSyncServant(stateQueryService service.StateQueryService, blockService service.BlockService) SyncServant {
	syncservant := &syncServant{
		StateQueryService: stateQueryService,
		BlockService:      blockService,
	}
	return syncservant
}

type syncServant struct {
	service.StateQueryService
	service.BlockService
	comp state.CompositionState
	account.AccountState
	chainState chain.ChainState
	epochState epoch.EpochState
	node2.NodeState
	network.Network
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

func (syncServ *syncServant) SetBlock(block *tpchaintypes.Block) error {
	panic("implement me")
}

func (syncServ *syncServant) GetChainRoot() ([]byte, error) {
	return syncServ.chainState.GetChainRoot()
}
func (syncServ *syncServant) GetRoundStateRoot() ([]byte, error) {
	return syncServ.epochState.GetRoundStateRoot()
}
func (syncServ *syncServant) SetLatestEpoch(epoch *tpcmm.EpochInfo) error {
	return syncServ.epochState.SetLatestEpoch(epoch)
}
