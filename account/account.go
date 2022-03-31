package account

import (
	"math/big"
	tpcrtypes "github.com/TopiaNetwork/topia/crypt/types"
)

type Address string

type Account struct {
	Addr tpcrtypes.Address
	Name string
	Nonce   uint64
	Balance *big.Int
}

