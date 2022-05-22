package transactionpool

import (
	txpooli "github.com/TopiaNetwork/topia/transaction_pool/interface"
	"io/ioutil"
)

func (pool *transactionPool) LoadConfig() (conf *txpooli.TransactionPoolConfig, error error) {
	data, err := ioutil.ReadFile(pool.config.PathConfigFile)
	if err != nil {
		return nil, err
	}
	config := &conf
	err = pool.marshaler.Unmarshal(data, &config)
	if err != nil {
		return nil, err
	}
	return *config, nil
}

func (pool *transactionPool) SetTxPoolConfig(conf txpooli.TransactionPoolConfig) {
	conf = (conf).Check()
	pool.config = conf
	return
}

func (pool *transactionPool) SaveConfig() error {
	//pool.log.Info("saving tx pool config")
	conf, err := pool.marshaler.Marshal(pool.config)
	if err != nil {
		return err
	}

	err = ioutil.WriteFile(pool.config.PathConfigFile, conf, 0664)
	if err != nil {
		return err
	}
	return nil
}

func (pool *transactionPool) loadConfig(path string) {
	if path != "" {
		if con, err := pool.LoadConfig(); err != nil {
			pool.log.Warnf("Failed to load txPool configs", "err", err)
		} else {
			pool.config = *con
		}
	}
}
