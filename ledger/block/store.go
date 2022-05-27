package block

import (
	"fmt"
	"github.com/TopiaNetwork/topia/chain/types"
	tplgtypes "github.com/TopiaNetwork/topia/ledger/types"
	"github.com/TopiaNetwork/topia/transaction/basic"
	"os"
	"path/filepath"
	"syscall"
	"unsafe"

	"github.com/TopiaNetwork/topia/ledger/backend"
	tplog "github.com/TopiaNetwork/topia/log"
	tplogcmm "github.com/TopiaNetwork/topia/log/common"
	"launchpad.net/gommap"
)

const defaultMaxFileSize = 1 << 28        // 假设文件最大为 128M
const defaultMemMapSize = 64 * (1 << 20)  // 内存映射大小64M

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
	// panic("implement me")

	var filename = "test.txt"
	var f *os.File
	if checkFileIsExist(filename) {
		f, _ = os.OpenFile(filename, os.O_APPEND, 0666)
		fmt.Println("not exist")
	} else {
		f, _ = os.Create(filename) //创建文件
		fmt.Println("not exist")
	}
	defer f.Close()

	var length = block.XXX_sizecache;
	var offset = 0;

	f.Write(make([]byte, length))
	fd := int(f.Fd())

	b, err := syscall.Mmap(fd, 5, length, syscall.PROT_READ|syscall.PROT_WRITE, syscall.MAP_SHARED)
	if err != nil {
		//panic(err)
	}

	b[offset-1] = 'x';

	err = syscall.Munmap(b)
	if err != nil {
		//panic(err)
	}
}

func (store *blockStore) GetBlockByNumber(blockNum types.BlockNum) (*types.Block, error) {
	//TODO implement me
	// panic("implement me")
	buf := make([]byte, *types.Block)
	if _, err := df.File.ReadAt(buf, offset); err != nil {
		return nil, err
	}

	item, err := DecodeDbItemHdr(buf)
	if err != nil {
		return nil, err
	}

	offset += DbItemHdrSize
	if item.KeySize > 0 {
		key := make([]byte, item.KeySize)
		if _, err := df.File.ReadAt(key, offset); err != nil {
			return nil, err
		}
		item.Key = key
	}

	offset += int64(item.KeySize)
	if item.ValSize > 0 {
		value := make([]byte, item.ValSize)
		if _, err := df.File.ReadAt(value, offset); err != nil {
			return nil, err
		}
		item.Val = value
	}
	return item, nil

}

func (store *blockStore) GetBlocksIterator(startBlockNum types.BlockNum) (tplgtypes.ResultsIterator, error) {
	//TODO implement me
	// panic("implement me")


}

func (store *blockStore) TxIDExists(txID basic.TxID) (bool, error) {
	//TODO implement me
	// panic("implement me")
	if store.GetTransactionByID(){

	}

}

func (store *blockStore) GetTransactionByID(txID basic.TxID) (*basic.Transaction, error) {
	//TODO implement me
	txId, _ := tx.HashHex()
}

func (store *blockStore) GetBlockByHash(blockHash []byte) (*types.Block, error) {
	//TODO implement me
	// panic("implement me")
	if blockHash == nil {

	}
	block, err := store.fileMgr.retrieveBlockByHash(blockHash)
	if err != nil {

	}


	return block,err
}

func (store *blockStore) GetBlockByTxID(txID string) (*types.Block, error) {
	//TODO implement me
	// panic("implement me")
	block, err := vledger.GetBlockByTxID(txID)
	if err != nil {
	}

	bytes, err := protoutil.Marshal(block)
	if err != nil {
		return shim.Error(err.Error())
	}

	return shim.Success(bytes)
	
}
