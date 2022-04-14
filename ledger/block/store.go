package block

import (
	"github.com/TopiaNetwork/topia/chain/types"
	tplgtypes "github.com/TopiaNetwork/topia/ledger/types"
	"github.com/TopiaNetwork/topia/transaction/basic"
	"path/filepath"

	"github.com/TopiaNetwork/topia/ledger/backend"
	tplog "github.com/TopiaNetwork/topia/log"
	tplogcmm "github.com/TopiaNetwork/topia/log/common"
)

type BlockStore interface {
	CommitBlock(block *types.Block) error

	GetBlockByNumber(blockNum types.BlockNum) (*types.Block, error)

	GetBlocksIterator(startBlockNum types.BlockNum) (tplgtypes.ResultsIterator, error)

	TxIDExists(txID basic.TxID) (bool, error)

	GetTransactionByID(txID basic.TxID) (*basic.Transaction, error)

	GetBlockByHash(blockHash []byte) (*types.Block, error)

	GetBlockByTxID(txID string) (*types.Block, error)
}

type blockStore struct {
	log     tplog.Logger
	fileMgr *fileManager
}

func NewBlockStore(log tplog.Logger, rootPath string, backendType backend.BackendType) BlockStore {
	bsLog := tplog.CreateModuleLogger(tplogcmm.InfoLevel, "blockStore", log)
	backend := backend.NewBackend(backendType, bsLog, filepath.Join(rootPath, "blockstore"), "blockstore")
	index := newBlockIndex(bsLog, backend)
	fileManager := newFileManager(bsLog, rootPath, index, backend)

	return &blockStore{
		bsLog,
		fileManager,
	}
}

func (store *blockStore) CommitBlock(block *types.Block) error {
	//TODO implement me
	panic("implement me")
}

func (store *blockStore) GetBlockByNumber(blockNum types.BlockNum) (*types.Block, error) {
	//TODO implement me
	panic("implement me")
}

func (store *blockStore) GetBlocksIterator(startBlockNum types.BlockNum) (tplgtypes.ResultsIterator, error) {
	//TODO implement me
	panic("implement me")
}

func (store *blockStore) TxIDExists(txID basic.TxID) (bool, error) {
	//TODO implement me
	panic("implement me")
}

func (store *blockStore) GetTransactionByID(txID basic.TxID) (*basic.Transaction, error) {
	//TODO implement me
	panic("implement me")
}

func (store *blockStore) GetBlockByHash(blockHash []byte) (*types.Block, error) {
	//TODO implement me
	panic("implement me")
}

func (store *blockStore) GetBlockByTxID(txID string) (*types.Block, error) {
	//TODO implement me
	panic("implement me")
}
