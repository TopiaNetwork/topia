package key_store

import tpcrtypes "github.com/TopiaNetwork/topia/crypt/types"

/*
KeyStore needs to store 2 kinds of items as below.

Address item:
	key: 	user account's address
	value: 	KeyItem

WalletEnable item:
	key: 	indicator of WalletEnable, such as string "wallet_Enable"
	value: 	WalletEnable state, enabled or disabled

*/
type KeyStore interface {
	SetAddr(addr string, item KeyItem) error
	GetAddr(addr string) (KeyItem, error)

	SetEnable(set bool) error
	GetEnable() (bool, error)

	Keys() (addrs []string, err error) // Show all addrs stored in wallet.

	Remove(key string) error
}

type KeyItem struct {
	CryptType tpcrtypes.CryptType  `json:"cryptType"`
	Seckey    tpcrtypes.PrivateKey `json:"seckey"`
}

const (
	EnableKey = "wallet_Enable"
	Enabled   = "wallet_Enabled"
	Disabled  = "wallet_Disabled"

	DefaultAddrKey = "default_Addr"
)
