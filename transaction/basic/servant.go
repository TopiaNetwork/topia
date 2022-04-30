package basic

import (
	lru "github.com/hashicorp/golang-lru"
	"math/big"

	"github.com/TopiaNetwork/topia/account"
	tpchaintypes "github.com/TopiaNetwork/topia/chain/types"
	"github.com/TopiaNetwork/topia/configuration"
	tpcrt "github.com/TopiaNetwork/topia/crypt"
	tpcrtypes "github.com/TopiaNetwork/topia/crypt/types"
	"github.com/TopiaNetwork/topia/currency"
	tplog "github.com/TopiaNetwork/topia/log"
	tpnet "github.com/TopiaNetwork/topia/network"
	stateaccount "github.com/TopiaNetwork/topia/state/account"
	statechain "github.com/TopiaNetwork/topia/state/chain"
)

type TansactionServant interface {
	ChainID() tpchaintypes.ChainID

	NetworkType() tpnet.NetworkType

	GetCryptService(log tplog.Logger, cryptType tpcrtypes.CryptType) (tpcrt.CryptService, error)

	GetGasConfig() *configuration.GasConfiguration

	GetChainConfig() *configuration.ChainConfiguration

	GetNonce(addr tpcrtypes.Address) (uint64, error)

	GetBalance(addr tpcrtypes.Address, symbol currency.TokenSymbol) (*big.Int, error)

	GetAccount(addr tpcrtypes.Address) (*account.Account, error)

	AddAccount(acc *account.Account) error

	UpdateNonce(addr tpcrtypes.Address, nonce uint64) error

	UpdateBalance(addr tpcrtypes.Address, symbol currency.TokenSymbol, value *big.Int) error
}

func NewTansactionServant(chainState statechain.ChainState, accountState stateaccount.AccountState) TansactionServant {
	return &tansactionServant{
		chainState,
		accountState,
	}
}

func NewTansactionServantSimulate(chainState statechain.ChainState, accountState stateaccount.AccountState) TansactionServant {
	ts := &tansactionServant{
		chainState,
		accountState,
	}

	lruCache, _ := lru.New(50)
	return &tansactionServantSimulate{
		tansactionServant: ts,
		cache:             lruCache,
	}
}

type tansactionServant struct {
	statechain.ChainState
	stateaccount.AccountState
}

type tansactionServantSimulate struct {
	*tansactionServant
	cache *lru.Cache
}

func (ts *tansactionServant) GetCryptService(log tplog.Logger, cryptType tpcrtypes.CryptType) (tpcrt.CryptService, error) {
	return tpcrt.CreateCryptService(log, cryptType), nil
}

func (ts *tansactionServant) GetGasConfig() *configuration.GasConfiguration {
	return configuration.GetConfiguration().GasConfig
}

func (ts *tansactionServant) GetChainConfig() *configuration.ChainConfiguration {
	return configuration.GetConfiguration().ChainConfig
}

func (tss *tansactionServantSimulate) AddAccount(acc *account.Account) error {
	return nil
}

func (tss *tansactionServantSimulate) UpdateNonce(addr tpcrtypes.Address, nonce uint64) error {
	return nil
}

func (tss *tansactionServantSimulate) UpdateBalance(addr tpcrtypes.Address, symbol currency.TokenSymbol, value *big.Int) error {
	return nil
}
