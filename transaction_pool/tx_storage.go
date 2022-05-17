package transactionpool

import (
	"encoding/json"
	"io/ioutil"
	"time"

	txbasic "github.com/TopiaNetwork/topia/transaction/basic"
)

func (pool *transactionPool) SaveLocalTxs(category txbasic.TransactionCategory) error {

	locals, err := json.Marshal(pool.allTxsForLook.getLocalKeyTxFromAllTxsLookupByCategory(category))
	if err != nil {
		return err
	}
	err = ioutil.WriteFile(pool.config.PathLocal[category], locals, 0664)
	if err != nil {
		return err
	}
	return nil
}

func (pool *transactionPool) loadLocal(category txbasic.TransactionCategory, nofile bool, path string) {
	if !nofile && path != "" {
		if err := pool.LoadLocalTxs(category); err != nil {
			pool.log.Warnf("Failed to load local transaction from stored file", "err", err)
		}
	}
}
func (pool *transactionPool) LoadLocalTxs(category txbasic.TransactionCategory) error {
	data, err := ioutil.ReadFile(pool.config.PathLocal[category])
	if err != nil {
		return nil
	}
	var locals map[string]*txbasic.Transaction
	err = json.Unmarshal(data, &locals)
	if err != nil {
		return err
	}
	for _, tx := range locals {
		pool.AddLocal(tx)
	}
	return nil
}

type remoteTxs struct {
	Txs                 map[txbasic.TxID]*txbasic.Transaction
	ActivationIntervals map[txbasic.TxID]time.Time
	HeightIntervals     map[txbasic.TxID]uint64
	TxHashCategorys     map[txbasic.TxID]txbasic.TransactionCategory
}

func (pool *transactionPool) SaveRemoteTxs(category txbasic.TransactionCategory) error {

	var remotetxs = &remoteTxs{
		Txs:                 pool.allTxsForLook.getRemoteMapTxsLookupByCategory(category),
		ActivationIntervals: pool.ActivationIntervals.getAll(),
		HeightIntervals:     pool.HeightIntervals.getAll(),
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

func (pool *transactionPool) loadRemote(category txbasic.TransactionCategory, nofile bool, path string) {
	if !nofile && path != "" {
		if err := pool.LoadRemoteTxs(category); err != nil {
			pool.log.Warnf("Failed to load remote transactions", "err", err)
		}
	}
}
func (pool *transactionPool) LoadRemoteTxs(category txbasic.TransactionCategory) error {
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
	for txId, height := range remotetxs.HeightIntervals {
		pool.HeightIntervals.setTxHeight(txId, height)
	}
	for txId, TxHashCategory := range remotetxs.TxHashCategorys {
		pool.TxHashCategory.setHashCat(txId, TxHashCategory)
	}
	return nil
}
