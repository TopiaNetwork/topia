package block

import (
	tptypes "github.com/TopiaNetwork/topia/common/types"
	"path/filepath"

	"github.com/TopiaNetwork/topia/ledger/backend"
	tplog "github.com/TopiaNetwork/topia/log"
	tplogcmm "github.com/TopiaNetwork/topia/log/common"
)

type BlockStore struct {
	log     tplog.Logger
	fileMgr *fileManager
}

func NewBlockStore(log tplog.Logger, rootPath string, backendType backend.BackendType) *BlockStore {
	bsLog := tplog.CreateModuleLogger(tplogcmm.InfoLevel, "BlockStore", log)
	backend := backend.NewBackend(backendType, bsLog, filepath.Join(rootPath, "blockstore"), "blockstore")
	index := newBlockIndex(bsLog, backend)
	fileManager := newFileManager(bsLog, rootPath, index, backend)

	return &BlockStore{
		bsLog,
		fileManager,
	}
}

func (store *BlockStore) AddBlock(block *tptypes.Block) error {
	panic("implement me")
}

func (store *BlockStore) RetrieveBlockByHash(blockHash []byte) (*tptypes.Block, error) {
	panic("implement me")
}

func (store *BlockStore) RetrieveBlockByNumber(blockNum uint64) (*tptypes.Block, error) {
	panic("implement me")
}

func (store *BlockStore) TxIDExists(txID string) (bool, error) {
	panic("implement me")
}

func (store *BlockStore) RetrieveBlockByTxID(txID string) (*tptypes.Block, error) {
	panic("implement me")
}
