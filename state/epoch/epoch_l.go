package epoch

import (
	"errors"
	"unsafe"

	"go.uber.org/atomic"

	tpcmm "github.com/TopiaNetwork/topia/common"
)

var (
	latestEpochMap map[string]*atomic.UnsafePointer
)

func init() {
	latestEpochMap = make(map[string]*atomic.UnsafePointer)
}

func updateLatestEpoch(ledgerID string, eInfo *tpcmm.EpochInfo) {
	if latestEpochMap[ledgerID] != nil {
		latestEpochMap[ledgerID].Swap(unsafe.Pointer(eInfo))
	} else {
		latestEpochMap[ledgerID] = atomic.NewUnsafePointer(unsafe.Pointer(eInfo))
	}
}

func GetLatestEpoch(ledgerID string) (*tpcmm.EpochInfo, error) {
	if latestEpochMap[ledgerID] != nil {
		return (*tpcmm.EpochInfo)(latestEpochMap[ledgerID].Load()), nil
	}

	return nil, errors.New("Nil latest epoch cache")
}
