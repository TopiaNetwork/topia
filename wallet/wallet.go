package wallet

import tpcrtypes "github.com/TopiaNetwork/topia/crypt/types"

type Wallet interface {
	Create(cryptType tpcrtypes.CryptType) (tpcrtypes.Address, error)

	Recovery(cryptType tpcrtypes.CryptType, mnemonic string, passphrase string) (tpcrtypes.Address, error)

	Import(privKey tpcrtypes.PrivateKey) (tpcrtypes.Address, error)

	Delete(tpcrtypes.Address) error

	SetDefault(tpcrtypes.Address) error

	Default() (tpcrtypes.Address, error)

	Export(tpcrtypes.Address) (tpcrtypes.PrivateKey, error)

	Sign(addr tpcrtypes.Address, msg []byte) (tpcrtypes.SignatureInfo, error)

	Has(tpcrtypes.Address) (bool, error)

	List() ([]tpcrtypes.Address, error)

	Lock(addr tpcrtypes.Address, lock bool) error

	IsLocked(addr tpcrtypes.Address) (bool, error)

	Enable(set bool) error

	IsEnable() (bool, error)
}
