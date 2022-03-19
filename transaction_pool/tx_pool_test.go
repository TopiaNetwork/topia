package transactionpool

import (
	"github.com/TopiaNetwork/topia/account"
	"github.com/TopiaNetwork/topia/codec"
	"github.com/TopiaNetwork/topia/transaction"
	"github.com/golang/mock/gomock"
	"testing"
	"time"
)

var Tx1 = settransaction()
var txpoolcfg = settxpoolconfig()

func settransaction() *transaction.Transaction {
	tx := &transaction.Transaction{
		FromAddr:   []byte{0x21, 0x23, 0x43, 0x53, 0x23, 0x34, 0x21, 0x12, 0x42, 0x12, 0x43},
		TargetAddr: []byte{0x23, 0x34, 0x34}, Version: 1, ChainID: []byte{0x01, 0x32},
		Nonce: 1, Value: []byte{0x12, 0x32}, GasPrice: 10000,
		GasLimit: 100, Data: []byte{0x32, 0x32, 0x32},
		Signature: []byte{0x32, 0x23, 0x42}, Options: 321, Time: time.Now()}
	return tx
}
func settxpoolconfig() *TransactionPoolConfig {
	conf := &TransactionPoolConfig{
		chain: nil,
		Locals: []account.Address{
			account.Address("0x2fadf9192731273"),
			account.Address("0x3fadfa123123123"),
			account.Address("0x3131313asa1daaf")},
		NoLocalFile:           false,
		NoRemoteFile:          false,
		NoConfigFile:          false,
		PathLocal:             "",
		PathRemote:            "",
		PathConfig:            "",
		ReStoredDur:           123,
		GasPriceLimit:         123,
		PendingAccountSlots:   123,
		PendingGlobalSlots:    123,
		QueueMaxTxsAccount:    123,
		QueueMaxTxsGlobal:     123,
		LifetimeForTx:         123,
		DurationForTxRePublic: 123,
	}
	return conf
}

func TestValidateTx(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()
	servant := NewMockTransactionPoolServant(ctrl)
	txpoolcfg.chain = servant
	log := NewMockLogger(ctrl)
	pool := NewTransactionPool(*txpoolcfg, 1, log, codec.CodecType(1))
	tx := Tx1
	if err := pool.ValidateTx(tx, true); err != ErrOversizedData {
		t.Error("expected", ErrOversizedData, "got", err)
	}
}
