package account

import (
	tpcrtypes "github.com/TopiaNetwork/topia/crypt/types"
)

type Account struct {
	Addr tpcrtypes.Address
	Name string
}
