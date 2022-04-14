package account

import (
	"github.com/TopiaNetwork/topia/currency"
	"math/big"

	tpcrtypes "github.com/TopiaNetwork/topia/crypt/types"
)

type Account struct {
	Addr    tpcrtypes.Address
	Nonce   uint64
	Name    string
	Balance map[currency.TokenSymbol]*big.Int
}
