package block

import (
	"fmt"
	tptypes "github.com/TopiaNetwork/topia/chain/types"

	"github.com/TopiaNetwork/topia/ledger/backend"
	tplog "github.com/TopiaNetwork/topia/log"
)

type locPointer struct {
	offset      int
	bytesLength int
}

func (lp *locPointer) String() string {
	return fmt.Sprintf("offset=%d, bytesLength=%d",
		lp.offset, lp.bytesLength)
}

// fileLocPointer
type fileLocPointer struct {
	fileSuffixNum int
	locPointer
}

type txindexInfo struct {
	txID string
	loc  *locPointer
}

type blockIndexInfo struct {
	blockNum  tptypes.BlockNum
	blockHash tptypes.BlockHash
	flp       *fileLocPointer
	txOffsets []*txindexInfo
}

type blockIndex struct {
	log     tplog.Logger
	backend backend.Backend
}

func newBlockIndex(log tplog.Logger, backend backend.Backend) *blockIndex {
	return &blockIndex{
		log:     log,
		backend: backend,
	}
}

func (bi *blockIndex) buildIndex(biInfo *blockIndexInfo) error {
	panic("implement me")
}

func (bi *blockIndex) getBlockLocByHash(blockHash []byte) (*fileLocPointer, error) {
	panic("implement me")
}

func (index *blockIndex) getBlockLocByBlockNum(blockNum uint64) (*fileLocPointer, error) {
	panic("implement me")
}

func (index *blockIndex) getTxLoc(txID string) (*fileLocPointer, error) {
	panic("implement me")
}

func (index *blockIndex) getBlockLocByTxID(txID string) (*fileLocPointer, error) {
	panic("implement me")
}

func (index *blockIndex) txIDExists(txID string) (bool, error) {
	panic("implement me")
}

func (index *blockIndex) getTXLocByBlockNumTranNum(blockNum uint64, tranNum uint64) (*fileLocPointer, error) {
	panic("implement me")
}
