package account

import "math/big"

type Address string

type Account struct {
	Addr    Address
	Name    string
	Nonce   uint64
	Balance *big.Int
}
