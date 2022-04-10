package transactionpool

import (
	"encoding/json"
	"io/ioutil"

	"github.com/TopiaNetwork/topia/transaction/basic"
)

func (pool *transactionPool) SaveLocalTxs(category basic.TransactionCategory) error {

	locals, err := json.Marshal(pool.allTxsForLook.getAllTxsLookupByCategory(category).GetAllLocalKeyTxs())
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
