package wallet

import tpcrtypes "github.com/TopiaNetwork/topia/crypt/types"

/*
keyStore needs to store 2 kinds of items as below.

Address item:
	key: 	user account's address
	value: 	keyItem

WalletEnable item:
	key: 	indicator of WalletEnable, such as string "wallet_Enable"
	value: 	WalletEnable state, enabled or disabled

*/
type keyStore interface {
	SetAddr(addr string, item keyItem) error
	GetAddr(addr string) (keyItem, error)

	SetEnable(set bool) error
	GetEnable() (bool, error)

	Keys() (addrs []string, err error) // Show all addrs stored in wallet.

	Remove(key string) error
}

type keyItem struct {
	CryptType tpcrtypes.CryptType  `json:"cryptType"`
	Seckey    tpcrtypes.PrivateKey `json:"seckey"`
}

const (
	walletEnableKey = "wallet_Enable"
	walletEnabled   = "wallet_Enabled"
	walletDisabled  = "wallet_Disabled"

	defaultAddrKey = "default_Addr"
)
