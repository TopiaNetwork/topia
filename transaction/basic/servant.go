package basic

import (
	"github.com/TopiaNetwork/topia/chain/types"
	"github.com/TopiaNetwork/topia/currency"
	"math/big"

	"github.com/TopiaNetwork/topia/configuration"
	tpcrt "github.com/TopiaNetwork/topia/crypt"
	tpcrtypes "github.com/TopiaNetwork/topia/crypt/types"
	"github.com/TopiaNetwork/topia/log"
	tpnet "github.com/TopiaNetwork/topia/network"
	stateaccount "github.com/TopiaNetwork/topia/state/account"
	statechain "github.com/TopiaNetwork/topia/state/chain"
)

type TansactionServant interface {
	ChainID() types.ChainID

	NetworkType() tpnet.NetworkType

	GetNonce(addr tpcrtypes.Address) (uint64, error)

	GetBalance(addr tpcrtypes.Address, symbol currency.TokenSymbol) (*big.Int, error)

	GetCryptService(log log.Logger, cryptType tpcrtypes.CryptType) (tpcrt.CryptService, error)

	GetGasConfig() *configuration.GasConfiguration

	GetChainConfig() *configuration.ChainConfiguration
}

func NewTansactionServant(chainState statechain.ChainState, accountState stateaccount.AccountState) TansactionServant {
	return &tansactionServant{
		chainState,
		accountState,
	}
}

type tansactionServant struct {
	statechain.ChainState
	stateaccount.AccountState
}

func (ts *tansactionServant) GetCryptService(log log.Logger, cryptType tpcrtypes.CryptType) (tpcrt.CryptService, error) {
	return tpcrt.CreateCryptService(log, cryptType), nil
}

func (ts *tansactionServant) GetGasConfig() *configuration.GasConfiguration {
	return configuration.GetConfiguration().GasConfig
}

func (ts *tansactionServant) GetChainConfig() *configuration.ChainConfiguration {
	return configuration.GetConfiguration().ChainConfig
}
