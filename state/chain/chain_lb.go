package chain

import (
	"unsafe"

	"go.uber.org/atomic"

	tpchaintypes "github.com/TopiaNetwork/topia/chain/types"
	tplgss "github.com/TopiaNetwork/topia/ledger/state"
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

func GetLatestBlock(ledgerID string, stateStore tplgss.StateStore, lgUpdater LedgerStateUpdater) (*tpchaintypes.Block, error) {
	if latestBlockMap[ledgerID] != nil {
		return (*tpchaintypes.Block)(latestBlockMap[ledgerID].Load()), nil
	} else {
		cState := NewChainStore(ledgerID, stateStore, lgUpdater, 1)
		return cState.GetLatestBlock()
	}
}

func GetLatestBlockResult(ledgerID string, stateStore tplgss.StateStore, lgUpdater LedgerStateUpdater) (*tpchaintypes.BlockResult, error) {
	if latestBlockRSMap[ledgerID] != nil {
		return (*tpchaintypes.BlockResult)(latestBlockRSMap[ledgerID].Load()), nil
	} else {
		cState := NewChainStore(ledgerID, stateStore, lgUpdater, 1)
		return cState.GetLatestBlockResult()
	}
}
