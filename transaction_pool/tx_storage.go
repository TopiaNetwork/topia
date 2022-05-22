package transactionpool

import (
	"fmt"
	"io/ioutil"
	"time"

	txbasic "github.com/TopiaNetwork/topia/transaction/basic"
)

type remoteTxs struct {
	Txs map[txbasic.TxID]*txbasic.Transaction
}

type TxsInfo struct {
	ActivationIntervals map[txbasic.TxID]time.Time
	HeightIntervals     map[txbasic.TxID]uint64
	TxHashCategorys     map[txbasic.TxID]txbasic.TransactionCategory
}

func (pool *transactionPool) SaveRemoteTxs(category txbasic.TransactionCategory) error {

	var remotetxs = &remoteTxs{
		Txs: pool.allTxsForLook.getRemoteMapTxsLookupByCategory(category),
	}
	remotes, err := pool.marshaler.Marshal(remotetxs)
	if err != nil {
		return err
	}
	err = ioutil.WriteFile(pool.config.PathMapRemoteTxsByCategory[category], remotes, 0664)
	if err != nil {
		return err
	}
	return nil
}

func (pool *transactionPool) loadRemote(category txbasic.TransactionCategory, path string) {
	if path != "" {
		if err := pool.LoadRemoteTxs(category); err != nil {
			pool.log.Warnf("Failed to load remote transactions", "err", err)
		}
	}
}
func (pool *transactionPool) LoadRemoteTxs(category txbasic.TransactionCategory) error {
	data, err := ioutil.ReadFile(pool.config.PathMapRemoteTxsByCategory[category])
	if err != nil {
		return err
	}
	remotetxs := &remoteTxs{}
	err = pool.marshaler.Unmarshal(data, &remotetxs)
	if err != nil {
		return err
	}

	for _, tx := range remotetxs.Txs {
		txId, err := tx.TxID()
		fmt.Println("ready to load remote txid", txId)
		if err != nil {
			pool.log.Errorf("error Txid :", txId, err)
		}
		pool.AddRemote(tx)
	}
	return nil
}

func (pool *transactionPool) SaveTxsInfo() error {

	var txsInfo = &TxsInfo{
		ActivationIntervals: pool.ActivationIntervals.getAll(),
		HeightIntervals:     pool.HeightIntervals.getAll(),
		TxHashCategorys:     pool.TxHashCategory.getAll(),
	}
	info, err := pool.marshaler.Marshal(txsInfo)
	if err != nil {
		return err
	}
	err = ioutil.WriteFile(pool.config.PathTxsInfoFile, info, 0664)
	if err != nil {
		return err
	}
	return nil
}

func (pool *transactionPool) loadTxsInfo(path string) {
	if path != "" {
		if err := pool.LoadTxsInfo(); err != nil {
			pool.log.Warnf("Failed to load txs info", "err", err)
		}
	}
}
func (pool *transactionPool) LoadTxsInfo() error {
	data, err := ioutil.ReadFile(pool.config.PathTxsInfoFile)
	if err != nil {
		return err
	}
	txsInfo := &TxsInfo{}
	err = pool.marshaler.Unmarshal(data, &txsInfo)
	if err != nil {
		return err
	}

	for txId, ActivationInterval := range txsInfo.ActivationIntervals {
		pool.ActivationIntervals.setTxActiv(txId, ActivationInterval)
	}
	for txId, height := range txsInfo.HeightIntervals {
		pool.HeightIntervals.setTxHeight(txId, height)
	}
	for txId, TxHashCategory := range txsInfo.TxHashCategorys {
		pool.TxHashCategory.setHashCat(txId, TxHashCategory)
	}
	return nil
}
