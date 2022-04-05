package state

import (
	"bytes"
	"sync"
	"time"

	tpchaintypes "github.com/TopiaNetwork/topia/chain/types"
	"github.com/TopiaNetwork/topia/ledger"
	tplog "github.com/TopiaNetwork/topia/log"
)

/* CompositionStateBuilder is only for proposer and validator
   and new CompositionState will be created nonce received new added block to the chain.
   For executor, a new CompositionState will be created when a prepare packed txs created.
*/

var stateBuilder *CompositionStateBuilder
var once sync.Once

const Wait_StateStore_Time = 50 //ms

func GetStateBuilder() *CompositionStateBuilder {
	once.Do(func() {
		stateBuilder = &CompositionStateBuilder{}
	})

	return stateBuilder
}

type CompositionStateBuilder struct {
	sync               sync.RWMutex
	lastBlockHashBytes []byte
	lastBlock          *tpchaintypes.Block
	compStateCurrent   CompositionState
}

func (builder *CompositionStateBuilder) UpdateCompositionStateWhenNewBlockAdded(log tplog.Logger, ledger ledger.Ledger, block *tpchaintypes.Block) error {
	builder.sync.Lock()
	defer builder.sync.Unlock()

	hashBytes, err := block.HashBytes()
	if err != nil {
		log.Panic("Can't get new added block hash bytes")
	}

	if builder.lastBlockHashBytes != nil {
		if bytes.Compare(hashBytes, builder.lastBlockHashBytes) == 0 {
			log.Warnf("Received the same new added block")
			return nil
		}
	}

	for i := 1; builder.compStateCurrent.PendingStateStore() > 0 && i <= 3; i++ {
		log.Warnf("Haven't been commited state store, need waiting for %d ms, no. %d ", Wait_StateStore_Time, i)
		time.Sleep(Wait_StateStore_Time * time.Millisecond)
	}

	builder.compStateCurrent = CreateCompositionState(log, ledger)

	builder.lastBlockHashBytes = hashBytes
	builder.lastBlock = block

	return nil
}

func (builder *CompositionStateBuilder) CompositionState() CompositionState {
	builder.sync.RLock()
	defer builder.sync.RUnlock()

	return builder.compStateCurrent
}
