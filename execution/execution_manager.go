package execution

import tx "github.com/TopiaNetwork/topia/transaction"

type executionManager struct {
	executionMap map[tx.TransactionType]tx.Executable
}

func newExecutionManager() *executionManager {
	return &executionManager{
		executionMap: map[tx.TransactionType]tx.Executable{},
	}
}
