package block

import (
	tplgtypes "github.com/TopiaNetwork/topia/ledger/types"
	tx "github.com/TopiaNetwork/topia/transaction"
	"path/filepath"

	tpchaintypes "github.com/TopiaNetwork/topia/chain/types"
	"github.com/TopiaNetwork/topia/ledger/backend"
	tplog "github.com/TopiaNetwork/topia/log"
	tplogcmm "github.com/TopiaNetwork/topia/log/common"
)

type BlockStore interface {
	CommitBlock(block *tpchaintypes.Block) error

	GetBlockByNumber(blockNum tpchaintypes.BlockNum) (*tpchaintypes.Block, error)

	GetBlocksIterator(startBlockNum tpchaintypes.BlockNum) (tplgtypes.ResultsIterator, error)

	TxIDExists(txID tx.TxID) (bool, error)

	GetTransactionByID(txID tx.TxID) (*tx.Transaction, error)

	GetBlockByHash(blockHash []byte) (*tpchaintypes.Block, error)

	GetBlockByTxID(txID string) (*tpchaintypes.Block, error)
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

func (store *blockStore) GetBlockByNumber(blockNum tpchaintypes.BlockNum) (*tpchaintypes.Block, error) {
	//TODO implement me
	panic("implement me")
}

func (store *blockStore) GetBlocksIterator(startBlockNum tpchaintypes.BlockNum) (tplgtypes.ResultsIterator, error) {
	//TODO implement me
	panic("implement me")
}

func (store *blockStore) TxIDExists(txID tx.TxID) (bool, error) {
	//TODO implement me
	panic("implement me")
}

func (store *blockStore) GetTransactionByID(txID tx.TxID) (*tx.Transaction, error) {
	//TODO implement me
	panic("implement me")
}

func (store *blockStore) GetBlockByHash(blockHash []byte) (*tpchaintypes.Block, error) {
	//TODO implement me
	panic("implement me")
}

func (store *blockStore) GetBlockByTxID(txID string) (*tpchaintypes.Block, error) {
	//TODO implement me
	panic("implement me")
}
