package block

import (
	"path/filepath"

	tpchaintypes "github.com/TopiaNetwork/topia/chain/types"
	"github.com/TopiaNetwork/topia/ledger/backend"
	tplgtypes "github.com/TopiaNetwork/topia/ledger/types"
	tplog "github.com/TopiaNetwork/topia/log"
	tplogcmm "github.com/TopiaNetwork/topia/log/common"
	"github.com/TopiaNetwork/topia/transaction/basic"
)

type BlockStore interface {
	TxIDExists(txID basic.TxID) (bool, error)

	GetTransactionByID(txID basic.TxID) (*basic.Transaction, error)

	GetTransactionResultByID(txID basic.TxID) (*basic.TransactionResult, error)

	GetBlockByHash(blockHash tpchaintypes.BlockHash) (*tpchaintypes.Block, error)

	GetBlockByTxID(txID basic.TxID) (*tpchaintypes.Block, error)

	GetBlockByNumber(blockNum tpchaintypes.BlockNum) (*tpchaintypes.Block, error)

	GetBlocksIterator(startBlockNum tpchaintypes.BlockNum) (tplgtypes.ResultsIterator, error)

	CommitBlock(block *tpchaintypes.Block) error

	CommitBlockResult(block *tpchaintypes.BlockResult) error

	Rollback(blockNum tpchaintypes.BlockNum) error
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

func (store *blockStore) CommitBlock(block *tpchaintypes.Block) error {
	//TODO implement me
	panic("implement me")
}

func (store *blockStore) CommitBlockResult(block *tpchaintypes.BlockResult) error {
	//TODO implement me
	panic("implement me")
}

func (store *blockStore) Rollback(blockNum tpchaintypes.BlockNum) error {
	//TODO implement me
	panic("implement me")
}

func (store *blockStore) GetBlockByNumber(blockNum tpchaintypes.BlockNum) (*tpchaintypes.Block, error) {
	//TODO implement me
	panic("implement me")
}

func (store *blockStore) GetBlocksIterator(startBlockNum tpchaintypes.BlockNum) (tplgtypes.ResultsIterator, error) {
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

func (store *blockStore) GetTransactionResultByID(txID basic.TxID) (*basic.TransactionResult, error) {
	//TODO implement me
	panic("implement me")
}

func (store *blockStore) GetBlockByHash(blockHash tpchaintypes.BlockHash) (*tpchaintypes.Block, error) {
	//TODO implement me
	panic("implement me")
}

func (store *blockStore) GetBlockByTxID(txID basic.TxID) (*tpchaintypes.Block, error) {
	//TODO implement me
	panic("implement me")
}
