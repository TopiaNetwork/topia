package chain

import (
	"errors"
	"unsafe"

	"go.uber.org/atomic"

	tpchaintypes "github.com/TopiaNetwork/topia/chain/types"
)

var (
	latestBlockMap   map[string]*atomic.UnsafePointer
	latestBlockRSMap map[string]*atomic.UnsafePointer
)

func init() {
	latestBlockMap = make(map[string]*atomic.UnsafePointer)
	latestBlockRSMap = make(map[string]*atomic.UnsafePointer)
}

func updateLatestBlock(ledgerID string, b *tpchaintypes.Block) {
	if latestBlockMap[ledgerID] != nil {
		latestBlockMap[ledgerID].Swap(unsafe.Pointer(b))
	} else {
		latestBlockMap[ledgerID] = atomic.NewUnsafePointer(unsafe.Pointer(b))
	}
}

func updateLatestBlockRS(ledgerID string, brs *tpchaintypes.BlockResult) {
	if latestBlockRSMap[ledgerID] != nil {
		latestBlockRSMap[ledgerID].Swap(unsafe.Pointer(brs))
	} else {
		latestBlockRSMap[ledgerID] = atomic.NewUnsafePointer(unsafe.Pointer(brs))
	}
}

func GetLatestBlock(ledgerID string) (*tpchaintypes.Block, error) {
	if latestBlockMap[ledgerID] != nil {
		return (*tpchaintypes.Block)(latestBlockMap[ledgerID].Load()), nil
	}

	return nil, errors.New("Nil latest block cache")
}

func GetLatestBlockResult(ledgerID string) (*tpchaintypes.BlockResult, error) {
	if latestBlockRSMap[ledgerID] != nil {
		return (*tpchaintypes.BlockResult)(latestBlockRSMap[ledgerID].Load()), nil
	}

	return nil, errors.New("Nil latest block result cache")
}
