package execution

import (
	txaction "github.com/TopiaNetwork/topia/transaction/action"
	"github.com/TopiaNetwork/topia/transaction/universal"
)

type executionManager struct {
	executionMap map[universal.TransactionUniversalType]txaction.Executable
}

func newExecutionManager() *executionManager {
	return &executionManager{
		executionMap: map[universal.TransactionUniversalType]txaction.Executable{},
	}
}
