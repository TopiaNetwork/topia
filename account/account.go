package account

import (
	"math/big"

	"github.com/TopiaNetwork/topia/chain"

	tpcrtypes "github.com/TopiaNetwork/topia/crypt/types"
)

type Account struct {
	Addr    tpcrtypes.Address
	Name    string
	Nonce   uint64
	Balance map[chain.TokenSymbol]*big.Int
}
