package transactionpool

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"

	tpchaintypes "github.com/TopiaNetwork/topia/chain/types"
)

func Test_transactionPool_addTxsForBlocksRevert(t *testing.T) {
	pool1.TruncateTxPool()

	txs := txLocals[:20]
	pool1.addTxs(txs, true)
	pool1.addTxs(txRemotes[:20], false)
	assert.Equal(t, 18, len(pool1.GetRemoteTxs()))

	//NewBlock txs:txLocals[10:20]

	var blocks []*tpchaintypes.Block
	blocks = append(blocks, OldBlock)
	blocks = append(blocks, MidBlock)
	blocks = append(blocks, NewBlock)
	pool1.chanBlocksRevert <- blocks
	time.Sleep(5 * time.Second)

	assert.Equal(t, 54, len(pool1.GetRemoteTxs()))

}
