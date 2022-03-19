package transactionpool

import (
	"fmt"
	"github.com/TopiaNetwork/topia/account"
	"github.com/TopiaNetwork/topia/transaction"
	"testing"
)

var Tx1 = settransaction()
var txpoolcfg = settxpoolconfig()
var txpool = setupTxPool()

func settransaction() *transaction.Transaction {
	tx := &transaction.Transaction{
		FromAddr:   []byte{0x21, 0x23, 0x43, 0x53, 0x23, 0x34, 0x21, 0x12, 0x42, 0x12, 0x43},
		TargetAddr: []byte{0x23, 0x34, 0x34}, Version: 1, ChainID: []byte{0x01, 0x32, 0x46, 0x32, 0x32}, Nonce: 1, Value: []byte{0x12, 0x32}, GasPrice: 10000,
		GasLimit: 100, Data: []byte{0x32, 0x32, 0x32},
		Signature: []byte{0x32, 0x23, 0x42}, Options: 321}
	return tx
}
func settxpoolconfig() *TransactionPoolConfig {
	conf := &TransactionPoolConfig{
		chain: nil,
		Locals: []account.Address{
			account.Address("2fadf9192731273"),
			account.Address("3fadfa123123123"),
			account.Address("31313131dsaf")},
		NoLocalFile:           false,
		NoRemoteFile:          false,
		NoConfigFile:          false,
		PathLocal:             "",
		PathRemote:            "",
		PathConfig:            "",
		ReStoredDur:           0,
		GasPriceLimit:         0,
		PendingAccountSlots:   0,
		PendingGlobalSlots:    0,
		QueueMaxTxsAccount:    0,
		QueueMaxTxsGlobal:     0,
		LifetimeForTx:         0,
		DurationForTxRePublic: 0,
	}
	return conf
}

func setupTxPool() *transactionPool {
	conf := settxpoolconfig()

	newPool := NewTransactionPool(*conf, 1, nil, 1)
	return newPool
}

func TestValidateTx(t *testing.T) {
	//pool := txpool
	tx := Tx1
	txsize := tx.Size()
	fmt.Println(txsize)
}
