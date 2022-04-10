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
	pool.allTxsForLook.Mu.Lock()
	defer pool.allTxsForLook.Mu.Unlock()
	pool.ActivationIntervals.Mu.Lock()
	defer pool.ActivationIntervals.Mu.Unlock()
	pool.TxHashCategory.Mu.Lock()
	defer pool.TxHashCategory.Mu.Unlock()
	var remotetxs = &remoteTxs{}
	remotetxs.Txs = pool.allTxsForLook.all[category].remotes
	remotetxs.ActivationIntervals = pool.ActivationIntervals.activ
	remotetxs.TxHashCategorys = pool.TxHashCategory.hashCategoryMap
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
		pool.ActivationIntervals.Mu.RLock()
		defer pool.ActivationIntervals.Mu.RUnlock()
		pool.ActivationIntervals.activ[txId] = ActivationInterval
	}
	for txId, TxHashCategory := range remotetxs.TxHashCategorys {
		pool.TxHashCategory.Mu.RLock()
		defer pool.TxHashCategory.Mu.RUnlock()
		pool.TxHashCategory.hashCategoryMap[txId] = TxHashCategory
	}
	return nil
}
