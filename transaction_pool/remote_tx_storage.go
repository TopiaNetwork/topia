package transactionpool

import (
	"encoding/json"
	"io/ioutil"
	"time"

	"github.com/TopiaNetwork/topia/transaction/basic"
)

type remoteTxs struct {
	Txs                 map[string]*basic.Transaction
	ActivationIntervals map[string]time.Time
	TxHashCategorys     map[string]basic.TransactionCategory
}

func (pool *transactionPool) SaveRemoteTxs(category basic.TransactionCategory) error {

	var remotetxs = &remoteTxs{
		Txs:                 pool.allTxsForLook.getRemoteMapTxsLookupByCategory(category),
		ActivationIntervals: pool.ActivationIntervals.getAll(),
		TxHashCategorys:     pool.TxHashCategory.getAll(),
	}
	remotes, err := json.Marshal(remotetxs)
	if err != nil {
		return err
	}
	err = ioutil.WriteFile(pool.config.PathRemote[category], remotes, 0664)
	if err != nil {
		return err
	}
	return nil
}

func (pool *transactionPool) loadRemote(category basic.TransactionCategory, nofile bool, path string) {
	if !nofile && path != "" {
		if err := pool.LoadRemoteTxs(category); err != nil {
			pool.log.Warnf("Failed to load remote transactions", "err", err)
		}
	}
}
func (pool *transactionPool) LoadRemoteTxs(category basic.TransactionCategory) error {
	data, err := ioutil.ReadFile(pool.config.PathRemote[category])
	if err != nil {
		return nil
	}
	remotetxs := &remoteTxs{}
	err = json.Unmarshal(data, &remotetxs)
	if err != nil {
		return nil
	}

	for _, tx := range remotetxs.Txs {
		pool.AddRemote(tx)
	}
	for txId, ActivationInterval := range remotetxs.ActivationIntervals {
		pool.ActivationIntervals.setTxActiv(txId, ActivationInterval)
	}
	for txId, TxHashCategory := range remotetxs.TxHashCategorys {
		pool.TxHashCategory.setHashCat(txId, TxHashCategory)
	}
	return nil
}
