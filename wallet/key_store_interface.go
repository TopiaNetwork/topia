package wallet

import tpcrtypes "github.com/TopiaNetwork/topia/crypt/types"

/*
walletStore needs to store 3 kinds of items as below.

Address item:
	key: 	user account's address
	value: 	addrItem

WalletEnable item:
	key: 	indicator of WalletEnable, such as string "wallet_Enable"
	value: 	WalletEnable state, enabled or disabled

DefaultAddr item:
	key:	indicator of DefaultAddr, such as string "default_Addr"
	value:	user's default address of the Wallet
*/
type walletStore interface {
	Init(arg interface{}) error

	SetAddr(item addrItem) error
	GetAddr(addr string) (addrItem, error)
	List() (addrs []string, err error) // Show all addrs stored in wallet.

	SetWalletEnable(set bool) error
	GetWalletEnable() (bool, error)

	SetDefaultAddr(defaultAddr string) error
	GetDefaultAddr() (defaultAddr string, err error)

	RemoveItem(key string) error
}

type addrItem struct {
	Addr string

	AddrLocked bool
	Seckey     tpcrtypes.PrivateKey
	Pubkey     tpcrtypes.PublicKey
	CryptType  tpcrtypes.CryptType
}

const (
	walletEnableKey = "wallet_Enable"
	walletEnabled   = "wallet_Enabled"
	walletDisabled  = "wallet_Disabled"

	defaultAddrKey = "default_Addr"
)
