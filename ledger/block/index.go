package block

import (
	"fmt"
	tpchaintypes "github.com/TopiaNetwork/topia/chain/types"
	"github.com/pkg/errors"
	"os"

	"github.com/TopiaNetwork/topia/ledger/backend"
	tplog "github.com/TopiaNetwork/topia/log"
)

type locPointer struct {
	offset      int
	bytesLength int
}

const maxMapSize = 0x8000000000
const maxMmapStep = 1 << 30

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
	blockNum  tpchaintypes.BlockNum
	blockHash tpchaintypes.BlockHash
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
	// panic("implement me")

}

func (bi *blockIndex) getBlockLocByHash(blockHash []byte) (*fileLocPointer, error) {
	//panic("implement me")
	b, err := bi.backend.ReaderAt(blockHash)
	if err != nil {
		return nil, err
	}
	if b == nil {
		return nil, err
	}
	blkLoc := &fileLocPointer{}
	//if err := blkLoc.unmarshal(b); err != nil {
	//	return nil, err
	//}
	return blkLoc, nil
}

func (index *blockIndex) getBlockLocByBlockNum(blockNum uint64) (*fileLocPointer, error) {
	//panic("implement me")
	b, err := index.db.Get(constructBlockNumKey(blockNum))
	if err != nil {
		return nil, err
	}
	if b == nil {
		return nil, errors.Errorf("no such block number [%d] in index", blockNum)
	}
	blkLoc := &fileLocPointer{}
	if err := blkLoc.unmarshal(b); err != nil {
		return nil, err
	}
	return blkLoc, nil
}

func (index *blockIndex) getTxLoc(txID string) (*fileLocPointer, error) {
	// panic("implement me")
	v, _, err := index.getTxID(txID)
	if err != nil {
		return nil, err
	}
	txFLP := &fileLocPointer{}
	if err = txFLP.unmarshal(v.TxLocation); err != nil {
		return nil, err
	}
	return txFLP, nil
}

func (index *blockIndex) getBlockLocByTxID(txID string) (*fileLocPointer, error) {
	// panic("implement me")
}

func (index *blockIndex) txIDExists(txID string) (bool, error) {
	// panic("implement me")

}

func (index *blockIndex) getTXLocByBlockNumTranNum(blockNum uint64, tranNum uint64) (*fileLocPointer, error) {
	//panic("implement me")
}

func mmapSize(size int) (int, error) {
	// Double the size from 32KB until 1GB.
	for i := uint(15); i <= 30; i++ {
		if size <= 1<<i {
			return 1 << i, nil
		}
	}

	// Verify the requested size is not above the maximum allowed.
	if size > maxMapSize {
		return 0, fmt.Errorf("mmap too large")
	}

	// If larger than 1GB then grow by 1GB at a time.
	sz := int64(size)
	if remainder := sz % int64(maxMmapStep); remainder > 0 {
		sz += int64(maxMmapStep) - remainder
	}

	// Ensure that the mmap size is a multiple of the page size.
	// This should always be true since we're incrementing in MBs.
	pageSize := int64(os.Getpagesize())
	if (sz % pageSize) != 0 {
		sz = ((sz / pageSize) + 1) * pageSize
	}
}
