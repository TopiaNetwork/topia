package basic

import (
	"math/big"

	"github.com/hashicorp/golang-lru"

	"github.com/TopiaNetwork/topia/account"
	tpchaintypes "github.com/TopiaNetwork/topia/chain/types"
	"github.com/TopiaNetwork/topia/configuration"
	tpcrt "github.com/TopiaNetwork/topia/crypt"
	tpcrtypes "github.com/TopiaNetwork/topia/crypt/types"
	"github.com/TopiaNetwork/topia/currency"
	tplog "github.com/TopiaNetwork/topia/log"
	tpnet "github.com/TopiaNetwork/topia/network"
	"github.com/TopiaNetwork/topia/state"
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

	NetworkType() tpnet.NetworkType

	GetNonce(addr tpcrtypes.Address) (uint64, error)

	GetBalance(addr tpcrtypes.Address, symbol currency.TokenSymbol) (*big.Int, error)

	GetAccount(addr tpcrtypes.Address) (*account.Account, error)
}

type TransactionServant interface {
	TransactionServantBaseRead

	GetCryptService(log tplog.Logger, cryptType tpcrtypes.CryptType) (tpcrt.CryptService, error)

	GetGasConfig() *configuration.GasConfiguration

	GetChainConfig() *configuration.ChainConfiguration

	AddAccount(acc *account.Account) error

	UpdateNonce(addr tpcrtypes.Address, nonce uint64) error

	UpdateBalance(addr tpcrtypes.Address, symbol currency.TokenSymbol, value *big.Int) error

	SnapToMem(log tplog.Logger) TransactionServant
}

func NewTransactionServant(compState state.CompositionState) TransactionServant {
	return &transactionServant{
		ChainState:   compState,
		AccountState: compState,
	}
}

func NewTansactionServantSimulate(ts TransactionServant) TransactionServant {
	lruCache, _ := lru.New(50)
	return &transactionServantSimulate{
		TransactionServantBaseRead: ts,
		cache:                      lruCache,
	}
}

func NewTansactionServantSimulateByCompStateReadOnly(compStateReadOnly state.CompositionStateReadonly) TransactionServant {
	lruCache, _ := lru.New(50)
	return &transactionServantSimulate{
		TransactionServantBaseRead: compStateReadOnly,
		cache:                      lruCache,
	}
}

type transactionServant struct {
	statechain.ChainState
	stateaccount.AccountState
	compStateReadOnly state.CompositionStateReadonly
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

func (ts *transactionServant) SnapToMem(log tplog.Logger) TransactionServant {
	compStateMem := ts.compStateReadOnly.SnapToMem(log)
	return NewTransactionServant(compStateMem)
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

func (ts *transactionServantSimulate) SnapToMem(log tplog.Logger) TransactionServant {
	return ts
}

func (tss *transactionServantSimulate) AddAccount(acc *account.Account) error {
	return nil
}

func (tss *transactionServantSimulate) UpdateNonce(addr tpcrtypes.Address, nonce uint64) error {
	return nil
}

func (tss *transactionServantSimulate) UpdateBalance(addr tpcrtypes.Address, symbol currency.TokenSymbol, value *big.Int) error {
	return nil
}

func CreatTransactionServantSimulate(log tplog.Logger, ts TransactionServant) TransactionServant {
	switch CurrentTxServantPolicy {
	case TxServantPolicy_WR:
		txServant, ok := ts.(*transactionServant)
		if !ok {
			log.Panic("Can't create read&write tx servant because of ts is not transactionServant")
			return nil
		}
		return txServant.SnapToMem(log)
	case TxServantPolicy_RO:
		return NewTansactionServantSimulate(ts)
	default:
		log.Panicf("Inavlid tx servant policy %d", CurrentTxServantPolicy)
	}

	return nil
}

func CreatTransactionServantSimulateByCompStateReadOnly(log tplog.Logger, compStateReadOnly state.CompositionStateReadonly) TransactionServant {
	switch CurrentTxServantPolicy {
	case TxServantPolicy_WR:
		compState := compStateReadOnly.SnapToMem(log)
		return NewTransactionServant(compState)
	case TxServantPolicy_RO:
		return NewTansactionServantSimulateByCompStateReadOnly(compStateReadOnly)
	default:
		log.Panicf("Inavlid tx servant policy %d", CurrentTxServantPolicy)
	}

	return nil
}
