package account

import tpcrtypes "github.com/TopiaNetwork/topia/crypt/types"

var NativeContractAccount_Account *Account

func init() {
	NativeContractAccount_Account = NewDefaultAccount(tpcrtypes.NativeContractAddr_Account)
}
