package common

import (
	"errors"
	"math/big"

	tpacc "github.com/TopiaNetwork/topia/account"
	tpchaintypes "github.com/TopiaNetwork/topia/chain/types"
	tpcmm "github.com/TopiaNetwork/topia/common"
	"github.com/TopiaNetwork/topia/configuration"
	tpcrt "github.com/TopiaNetwork/topia/crypt"
	tpcrtypes "github.com/TopiaNetwork/topia/crypt/types"
	"github.com/TopiaNetwork/topia/currency"
	tplog "github.com/TopiaNetwork/topia/log"
	tpnet "github.com/TopiaNetwork/topia/network"
)

var (
	ErrInsufficientGas = errors.New("Insufficient gas")
)

type BaseServant interface {
	ChainID() tpchaintypes.ChainID

	NetworkType() tpnet.NetworkType

	GetCryptService(log tplog.Logger, cryptType tpcrtypes.CryptType) (tpcrt.CryptService, error)

	GetGasConfig() *configuration.GasConfiguration

	GetNonce(addr tpcrtypes.Address) (uint64, error)

	GetBalance(addr tpcrtypes.Address, symbol currency.TokenSymbol) (*big.Int, error)

	GetAccount(addr tpcrtypes.Address) (*tpacc.Account, error)

	AddAccount(acc *tpacc.Account) error

	UpdateAccount(account *tpacc.Account) error

	UpdateNonce(addr tpcrtypes.Address, nonce uint64) error

	UpdateBalance(addr tpcrtypes.Address, symbol currency.TokenSymbol, value *big.Int) error

	UpdateName(addr tpcrtypes.Address, name tpacc.AccountName) error
}

type VMServant interface {
	BaseServant

	GasUsedAccumulate(gasAdd uint64) error
}

type vmServant struct {
	BaseServant
	maxGasLimit uint64
	gasUsed     uint64
}

func NewVMServant(bServant BaseServant, maxGasLimit uint64) VMServant {
	return &vmServant{
		BaseServant: bServant,
		maxGasLimit: maxGasLimit,
	}
}

func (vs *vmServant) GasUsedAccumulate(gasAdd uint64) error {
	gasAdded, err := tpcmm.SafeAddUint64(vs.gasUsed, gasAdd)
	if err != nil {
		return err
	}

	if gasAdded > vs.maxGasLimit {
		return ErrInsufficientGas
	}

	return nil
}
