package validator

import (
	"math/big"

	"github.com/TopiaNetwork/topia/account"
	tpcrtypes "github.com/TopiaNetwork/topia/crypt/types"
)

type Validator struct {
	NodeID string
	Addr   account.Address
	PubKey tpcrtypes.PublicKey
	Weight *big.Int
}
