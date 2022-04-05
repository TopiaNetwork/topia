package account

import (
	tpcrtypes "github.com/TopiaNetwork/topia/crypt/types"
	"math/big"
)

type Account struct {
	Addr    tpcrtypes.Address
	Name    string
	Nonce   uint64
	Balance *big.Int
}
