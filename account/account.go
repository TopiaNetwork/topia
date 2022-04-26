package account

import (
	"fmt"
	"math/big"

	tpcrtypes "github.com/TopiaNetwork/topia/crypt/types"
	"github.com/TopiaNetwork/topia/currency"
)

type AccountVersion uint64

const (
	AccountVersion_Unknown AccountVersion = iota
	AccountVersion_V1
)

type Account struct {
	Addr     tpcrtypes.Address
	Nonce    uint64
	Name     AccountName
	Token    *AccountToken
	Balances map[currency.TokenSymbol]*big.Int
	CodeHash string
	Version  AccountVersion
}

func NewDefaultAccount(addr tpcrtypes.Address) *Account {
	accToken := &AccountToken{
		Permission: NewPermissionRoot(),
	}
	return &Account{
		Addr:     addr,
		Token:    accToken,
		Balances: make(map[currency.TokenSymbol]*big.Int),
		Version:  AccountVersion_V1,
	}
}

func NewContractControlAccount(addr tpcrtypes.Address, name AccountName, gasLimit uint64) *Account {
	if !name.IsValid() {
		panic("Invalid account name" + string(name))
	}

	accToken := &AccountToken{
		Permission: NewPermissionContractMethod(gasLimit),
	}

	return &Account{
		Addr:     addr,
		Name:     name,
		Token:    accToken,
		Balances: make(map[currency.TokenSymbol]*big.Int),
		Version:  AccountVersion_V1,
	}
}

func (a *Account) BindName(name string) {
	a.Name = AccountName(name)
}

func (a *Account) BalanceIncrease(symbol currency.TokenSymbol, value *big.Int) {
	symbolValue, ok := a.Balances[symbol]
	if !ok {
		symbolValue = new(big.Int)
	}

	a.Balances[symbol] = symbolValue.Add(symbolValue, value)
}

func (a *Account) BalanceDecrease(symbol currency.TokenSymbol, value *big.Int) error {
	symbolValue, ok := a.Balances[symbol]
	if !ok {
		return fmt.Errorf("Insufficient currency to decrease: %s, actual nil, value %v", symbol, value)
	}

	if symbolValue.Cmp(value) == -1 {
		return fmt.Errorf("Insufficient currency to decrease: %s, actual %v, value %v", symbol, symbolValue, value)
	}

	a.Balances[symbol] = symbolValue.Sub(symbolValue, value)

	return nil
}

func (a *Account) NonceIncrease() {
	a.Nonce++
}
