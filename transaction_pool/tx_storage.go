package transactionpool

import (
	"io/ioutil"
	"time"

	txbasic "github.com/TopiaNetwork/topia/transaction/basic"
	txpooli "github.com/TopiaNetwork/topia/transaction_pool/interface"
)

type TxsStorage struct {
	Categorys           []txbasic.TransactionCategory
	TxsLocal            []*txbasic.Transaction
	TxsRemote           []*txbasic.Transaction
	ActivationIntervals map[txbasic.TxID]time.Time
	HeightIntervals     map[txbasic.TxID]uint64
	TxHashCategorys     map[txbasic.TxID]txbasic.TransactionCategory
	Config              txpooli.TransactionPoolConfig
}

func (pool *transactionPool) SaveTxsData(path string) error {

	var txsStorge = &TxsStorage{
		Categorys:           pool.allTxsForLook.getAllCategory(),
		TxsLocal:            pool.allTxsForLook.getAllTxs(true),
		TxsRemote:           pool.allTxsForLook.getAllTxs(false),
		ActivationIntervals: pool.activationIntervals.getAll(),
		HeightIntervals:     pool.heightIntervals.getAll(),
		TxHashCategorys:     pool.txHashCategory.getAll(),
		Config:              pool.config,
	}
	txsData, err := pool.marshaler.Marshal(txsStorge)
	if err != nil {
		return err
	}
	err = ioutil.WriteFile(path, txsData, 0664)
	if err != nil {
		return err
	}
	return nil
}

func (pool *transactionPool) LoadTxsData(path string) {
	if path != "" {
		if err := pool.loadTxsData(path); err != nil {
			pool.log.Errorf("Failed to load transactions data", "err", err)
		} else {
			pool.log.Infof("loadTxsData from storage done")
		}
	} else {
		pool.log.Errorf("Failed to load transactions data: config.PathTxsStorge is nil")
	}
}
func (pool *transactionPool) loadTxsData(path string) error {
	data, err := ioutil.ReadFile(path)
	if err != nil {
		return err
	}
	txsData := &TxsStorage{}
	err = pool.marshaler.Unmarshal(data, &txsData)
	if err != nil {
		return err
	}

	pool.config = txsData.Config
	for _, cat := range txsData.Categorys {
		pool.newTxListStructs(cat)
	}
	pool.addTxs(txsData.TxsLocal, true)
	pool.addTxs(txsData.TxsRemote, false)
	for txId, ActivationInterval := range txsData.ActivationIntervals {
		pool.activationIntervals.setTxActiv(txId, ActivationInterval)
	}
	for txId, height := range txsData.HeightIntervals {
		pool.heightIntervals.setTxHeight(txId, height)
	}
	for txId, TxHashCategory := range txsData.TxHashCategorys {
		pool.txHashCategory.setHashCat(txId, TxHashCategory)
	}
	return nil
}
