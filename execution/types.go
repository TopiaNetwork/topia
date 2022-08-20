package execution

import (
	"github.com/lazyledger/smt"
	"sync"

	txbasic "github.com/TopiaNetwork/topia/transaction/basic"
)

type PackedTxs struct {
	StateVersion uint64
	TxRoot       []byte
	TxList       []*txbasic.Transaction
	sync         sync.Mutex
	treeTx       *smt.SparseMerkleTree
}

type PackedTxsResult struct {
	StateVersion uint64
	TxRSRoot     []byte
	TxsResult    []txbasic.TransactionResult
	sync         sync.Mutex
	treeTxRS     *smt.SparseMerkleTree
}
