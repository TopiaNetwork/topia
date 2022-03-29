package block

import (
	tpchaintypes "github.com/TopiaNetwork/topia/chain/types"
	"github.com/TopiaNetwork/topia/ledger/backend"
	tplog "github.com/TopiaNetwork/topia/log"
)

type blockfilesInfo struct {
	latestFileNumber   int
	latestFileSize     int
	noBlockFiles       bool
	lastPersistedBlock uint64
}

type fileManager struct {
	log      tplog.Logger
	rootPath string
	index    *blockIndex
	backend  backend.Backend
}

func newFileManager(log tplog.Logger, rootPath string, index *blockIndex, backend backend.Backend) *fileManager {
	return &fileManager{
		log:      log,
		rootPath: rootPath,
		index:    index,
		backend:  backend,
	}
}

func (fm *fileManager) moveToNextFile() {
	panic("implement me")
}

func (fm *fileManager) addBlock(block *tpchaintypes.Block) error {
	panic("implement me")
}

func (fm *fileManager) retrieveBlockByHash(blockHash tpchaintypes.BlockHash) (*tpchaintypes.Block, error) {
	panic("implement me")
}

func (fm *fileManager) retrieveBlockByNumber(blockNum tpchaintypes.BlockNum) (*tpchaintypes.Block, error) {
	panic("implement me")
}

func (fm *fileManager) retrieveBlockByTxID(txID string) (*tpchaintypes.Block, error) {
	panic("implement me")
}

func (mgr *fileManager) fetchBlock(lp *fileLocPointer) (*tpchaintypes.Block, error) {
	panic("implement me")
}

func (mgr *fileManager) fetchBlockBytes(lp *fileLocPointer) ([]byte, error) {
	panic("implement me")
}

func (mgr *fileManager) fetchRawBytes(lp *fileLocPointer) ([]byte, error) {
	panic("implement me")
}
