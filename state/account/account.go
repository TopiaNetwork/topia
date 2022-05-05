package account

import (
	"encoding/json"
	"fmt"
	"math/big"

	tpacc "github.com/TopiaNetwork/topia/account"
	tpcrtypes "github.com/TopiaNetwork/topia/crypt/types"
	"github.com/TopiaNetwork/topia/currency"
	tplgss "github.com/TopiaNetwork/topia/ledger/state"
)

const StateStore_Name = "account"

type AccountState interface {
	GetAccountRoot() ([]byte, error)

	GetAccountProof(addr tpcrtypes.Address) ([]byte, error)

	IsAccountExist(addr tpcrtypes.Address) bool

	GetAccount(addr tpcrtypes.Address) (*tpacc.Account, error)

	GetNonce(addr tpcrtypes.Address) (uint64, error)

	GetBalance(addr tpcrtypes.Address, symbol currency.TokenSymbol) (*big.Int, error)

	AddAccount(acc *tpacc.Account) error

	UpdateAccount(account *tpacc.Account) error

	UpdateNonce(addr tpcrtypes.Address, nonce uint64) error

	UpdateBalance(addr tpcrtypes.Address, symbol currency.TokenSymbol, value *big.Int) error

	UpdateName(addr tpcrtypes.Address, name tpacc.AccountName) error
}

type accountState struct {
	tplgss.StateStore
}

func NewAccountState(stateStore tplgss.StateStore) AccountState {
	stateStore.AddNamedStateStore(StateStore_Name)
	return &accountState{
		stateStore,
	}
}

func (as *accountState) GetAccountRoot() ([]byte, error) {
	return as.Root(StateStore_Name)
}

func (as *accountState) GetAccountProof(addr tpcrtypes.Address) ([]byte, error) {
	_, proof, err := as.GetState(StateStore_Name, addr.Bytes())

	return proof, err
}

func (as *accountState) IsAccountExist(addr tpcrtypes.Address) bool {
	isExist, _ := as.Exists(StateStore_Name, addr.Bytes())

	return isExist
}

func (as *accountState) GetAccount(addr tpcrtypes.Address) (*tpacc.Account, error) {
	accBytes, _, err := as.GetState(StateStore_Name, addr.Bytes())
	if err != nil {
		return nil, err
	}

	var acc tpacc.Account
	err = json.Unmarshal(accBytes, &acc)
	if err != nil {
		return nil, err
	}

	return &acc, nil
}

func (as *accountState) GetNonce(addr tpcrtypes.Address) (uint64, error) {
	acc, err := as.GetAccount(addr)
	if err != nil {
		return 0, err
	}

	return acc.Nonce, nil
}

func (as *accountState) GetBalance(addr tpcrtypes.Address, symbol currency.TokenSymbol) (*big.Int, error) {
	acc, err := as.GetAccount(addr)
	if err != nil {
		return nil, err
	}

	if balVal, ok := acc.Balances[symbol]; ok {
		return balVal, nil
	}

	return nil, fmt.Errorf("No responding symbol %s from addr %s", symbol, addr)
}

func (as *accountState) AddAccount(acc *tpacc.Account) error {
	if as.IsAccountExist(acc.Addr) {
		return fmt.Errorf("Have existed account from %s", acc.Addr)
	}

	accBytes, err := json.Marshal(acc)
	if err != nil {
		return err
	}

	return as.Put(StateStore_Name, acc.Addr.Bytes(), accBytes)
}

func (as *accountState) UpdateAccount(account *tpacc.Account) error {
	accBytes, err := json.Marshal(account)
	if err != nil {
		return err
	}

	return as.Update(StateStore_Name, account.Addr.Bytes(), accBytes)
}

func (as *accountState) UpdateNonce(addr tpcrtypes.Address, nonce uint64) error {
	acc, err := as.GetAccount(addr)
	if err != nil {
		return err
	}

	acc.Nonce = nonce

	return as.UpdateAccount(acc)
}

func (as *accountState) UpdateBalance(addr tpcrtypes.Address, symbol currency.TokenSymbol, value *big.Int) error {
	acc, err := as.GetAccount(addr)
	if err != nil {
		return err
	}

	if balVal, ok := acc.Balances[symbol]; ok {
		balVal.Set(value)
	} else {
		acc.Balances[symbol] = value
	}

	return as.UpdateAccount(acc)
}

func (as *accountState) UpdateName(addr tpcrtypes.Address, name tpacc.AccountName) error {
	acc, err := as.GetAccount(addr)
	if err != nil {
		return err
	}

	acc.Name = name

	return as.UpdateAccount(acc)
}

