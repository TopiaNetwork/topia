package basic

import (
	"github.com/TopiaNetwork/topia/codec"
	"github.com/TopiaNetwork/topia/common"
	"math/big"

	"github.com/hashicorp/golang-lru"

	tpacc "github.com/TopiaNetwork/topia/account"
	tpchaintypes "github.com/TopiaNetwork/topia/chain/types"
	"github.com/TopiaNetwork/topia/configuration"
	tpcrt "github.com/TopiaNetwork/topia/crypt"
	tpcrtypes "github.com/TopiaNetwork/topia/crypt/types"
	"github.com/TopiaNetwork/topia/currency"
	tplog "github.com/TopiaNetwork/topia/log"
	stateaccount "github.com/TopiaNetwork/topia/state/account"
	statechain "github.com/TopiaNetwork/topia/state/chain"
)

type TxServantPolicy byte

const (
	TxServantPolicy_Unknown TxServantPolicy = iota
	TxServantPolicy_WR
	TxServantPolicy_RO
)

var CurrentTxServantPolicy = TxServantPolicy_WR

type TransactionServantBaseRead interface {
	ChainID() tpchaintypes.ChainID

	NetworkType() common.NetworkType

	GetNonce(addr tpcrtypes.Address) (uint64, error)

	GetBalance(addr tpcrtypes.Address, symbol currency.TokenSymbol) (*big.Int, error)

	GetAccount(addr tpcrtypes.Address) (*tpacc.Account, error)

	GetMarshaler() codec.Marshaler

	GetTxPoolSize() int64

	GetLatestBlock() (*tpchaintypes.Block, error)
}

type TransactionServant interface {
	TransactionServantBaseRead

	GetCryptService(log tplog.Logger, cryptType tpcrtypes.CryptType) (tpcrt.CryptService, error)

	GetGasConfig() *configuration.GasConfiguration

	GetChainConfig() *configuration.ChainConfiguration

	AddAccount(acc *tpacc.Account) error

	UpdateAccount(account *tpacc.Account) error

	UpdateNonce(addr tpcrtypes.Address, nonce uint64) error

	UpdateBalance(addr tpcrtypes.Address, symbol currency.TokenSymbol, value *big.Int) error

	UpdateName(addr tpcrtypes.Address, name tpacc.AccountName) error
}

func NewTransactionServant(chainState statechain.ChainState,
	accountState stateaccount.AccountState,
	marshaler codec.Marshaler,
	txPoolSize func() int64) TransactionServant {
	return &transactionServant{
		ChainState:   chainState,
		AccountState: accountState,
		marshaler:    marshaler,
		txPoolSize:   txPoolSize,
	}
}

func NewTansactionServantSimulate(tsBaseRead TransactionServantBaseRead) TransactionServant {
	lruCache, _ := lru.New(50)
	return &transactionServantSimulate{
		TransactionServantBaseRead: tsBaseRead,
		cache:                      lruCache,
	}
}

type transactionServant struct {
	statechain.ChainState
	stateaccount.AccountState
	marshaler  codec.Marshaler
	txPoolSize func() int64
}

type transactionServantSimulate struct {
	TransactionServantBaseRead
	cache *lru.Cache
}

func (ts *transactionServant) GetCryptService(log tplog.Logger, cryptType tpcrtypes.CryptType) (tpcrt.CryptService, error) {
	return tpcrt.CreateCryptService(log, cryptType), nil
}

func (ts *transactionServant) GetGasConfig() *configuration.GasConfiguration {
	return configuration.GetConfiguration().GasConfig
}

func (ts *transactionServant) GetChainConfig() *configuration.ChainConfiguration {
	return configuration.GetConfiguration().ChainConfig
}

func (ts *transactionServant) GetMarshaler() codec.Marshaler {
	return ts.marshaler
}

func (ts *transactionServant) GetTxPoolSize() int64 {
	return ts.txPoolSize()
}

func (ts *transactionServantSimulate) GetCryptService(log tplog.Logger, cryptType tpcrtypes.CryptType) (tpcrt.CryptService, error) {
	return tpcrt.CreateCryptService(log, cryptType), nil
}

func (ts *transactionServantSimulate) GetGasConfig() *configuration.GasConfiguration {
	return configuration.GetConfiguration().GasConfig
}

func (ts *transactionServantSimulate) GetChainConfig() *configuration.ChainConfiguration {
	return configuration.GetConfiguration().ChainConfig
}

func (tss *transactionServantSimulate) AddAccount(acc *tpacc.Account) error {
	return nil
}

func (tss *transactionServantSimulate) UpdateAccount(account *tpacc.Account) error {
	return nil
}

func (tss *transactionServantSimulate) UpdateNonce(addr tpcrtypes.Address, nonce uint64) error {
	return nil
}

func (tss *transactionServantSimulate) UpdateBalance(addr tpcrtypes.Address, symbol currency.TokenSymbol, value *big.Int) error {
	return nil
}

func (tss *transactionServantSimulate) UpdateName(addr tpcrtypes.Address, name tpacc.AccountName) error {
	return nil
}
