package transactionpool

import (
	"encoding/json"
	"io/ioutil"

	"github.com/TopiaNetwork/topia/transaction/basic"
)

func (pool *transactionPool) SaveLocalTxs(category basic.TransactionCategory) error {
	locals, err := json.Marshal(pool.allTxsForLook[category].locals)
	if err != nil {
		return err
	}
	err = ioutil.WriteFile(pool.config.PathLocal[category], locals, 0664)
	if err != nil {
		return err
	}
	return nil
}

func (pool *transactionPool) loadLocal(category basic.TransactionCategory, nofile bool, path string) {
	if !nofile && path != "" {
		if err := pool.LoadLocalTxs(category); err != nil {
			pool.log.Warnf("Failed to load local transaction from stored file", "err", err)
		}
	}
}
func (pool *transactionPool) LoadLocalTxs(category basic.TransactionCategory) error {
	data, err := ioutil.ReadFile(pool.config.PathLocal[category])
	if err != nil {
		return nil
	}
	var locals map[string]*basic.Transaction
	err = json.Unmarshal(data, &locals)
	if err != nil {
		return err
	}
	for _, tx := range locals {
		pool.AddLocal(tx)
	}
	return nil
}

// AddLocals enqueues a batch of transactions into the pool if they are valid, marking the
// senders as a local ones, ensuring they go around the local pricing constraints.
//
// This method is used to add transactions from the RPC API and performs synchronous pool
// reorganization and event propagation.
func (pool *transactionPool) AddLocals(txs []*basic.Transaction) []error {
	return pool.addTxs(txs, !pool.config.NoLocalFile, true)
}

// AddLocal enqueues a single local transaction into the pool if it is valid. This is
// a convenience wrapper aroundd AddLocals.
func (pool *transactionPool) AddLocal(tx *basic.Transaction) error {
	errs := pool.AddLocals([]*basic.Transaction{tx})
	return errs[0]
}
