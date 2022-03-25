package account

import (
	"math/big"

	"github.com/TopiaNetwork/topia/chain"
	tpcrtypes "github.com/TopiaNetwork/topia/crypt/types"
	tplgss "github.com/TopiaNetwork/topia/ledger/state"
)

type AccountState interface {
	GetAccountRoot() ([]byte, error)

	GetNonce(addr tpcrtypes.Address) (uint64, error)

	GetBalance(symbol chain.TokenSymbol, addr tpcrtypes.Address) (*big.Int, error)
}

type accountState struct {
	tplgss.StateStore
}

func NewAccountState(stateStore tplgss.StateStore) AccountState {
	stateStore.AddNamedStateStore("account")
	return &accountState{
		stateStore,
	}
}

func (as *accountState) GetAccountRoot() ([]byte, error) {
	//TODO implement me
	panic("implement me")
}

func (as *accountState) GetNonce(addr tpcrtypes.Address) (uint64, error) {
	//TODO implement me
	panic("implement me")
}

func (as *accountState) GetBalance(symbol chain.TokenSymbol, addr tpcrtypes.Address) (*big.Int, error) {
	//TODO implement me
	panic("implement me")
}