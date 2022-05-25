package types

import (
	"errors"
	"math/big"
)

type Account struct {
	Addr    []byte
	Balance *big.Int
	Nonce   uint64
	Code    []byte
}

func (a *Account) GetAddr() string {
	return string(a.Addr)
}

func (a *Account) GetBalance() *big.Int {
	return a.Balance
}

func (a *Account) SetBalance(balance *big.Int) {
	a.Balance.Set(balance)
}

func (a *Account) AddBalance(diff *big.Int) {
	a.SetBalance(new(big.Int).Add(a.Balance, diff))
}

func (a *Account) SubBalance(diff *big.Int) error {
	if a.Balance.Cmp(diff) < 0 {
		return errors.New("balance not enough!")
	}
	a.SetBalance(new(big.Int).Sub(a.Balance, diff))
	return nil
}

func (a *Account) GetNonce() uint64 {
	return a.Nonce
}

func (a *Account) AddNonce() {
	a.Nonce++
}

func (a *Account) GetCode() []byte {
	return a.Code
}

func (a *Account) SetCode(code []byte) {
	a.Code = code
}
